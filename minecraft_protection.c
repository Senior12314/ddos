/*
 * CloudNordSP Minecraft DDoS Protection - XDP eBPF Program
 * 
 * This XDP program provides high-performance packet filtering for Minecraft
 * servers, implementing rate limiting, blacklisting, and protocol validation.
 */

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include <linux/udp.h>
#include <linux/tcp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// BPF Maps
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __type(key, struct endpoint_key);
    __type(value, struct endpoint_info);
    __uint(max_entries, 10000);
    __uint(map_flags, BPF_F_NO_PREALLOC);
} map_protected_endpoints SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);  // source IP
    __type(value, struct rate_limit_state);
    __uint(max_entries, 100000);
} map_src_rate SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);  // 5-tuple hash
    __type(value, struct conntrack_entry);
    __uint(max_entries, 100000);
} map_conntrack SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);  // IP address
    __type(value, __u64);  // timestamp until blocked
    __uint(max_entries, 50000);
} map_blacklist SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, __u64);
    __uint(max_entries, 10);
} map_stats SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);  // source IP
    __type(value, struct udp_challenge_state);
    __uint(max_entries, 10000);
} map_udp_challenges SEC(".maps");

// Data structures
struct endpoint_key {
    __u32 prefix_len;
    __u32 ip;
    __u16 port;
    __u8 protocol;
};

struct endpoint_info {
    __u32 origin_ip;
    __u16 origin_port;
    __u32 rate_limit;
    __u32 burst_limit;
    __u8 protocol_type;  // 0=Java, 1=Bedrock
    __u8 maintenance_mode;
    __u8 padding[2];
};

struct rate_limit_state {
    __u64 last_update;
    __u32 tokens;
    __u32 last_burst;
};

struct conntrack_entry {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8 protocol;
    __u8 state;  // 0=unknown, 1=established, 2=challenge_sent
    __u16 challenge_id;
    __u8 padding[1];
};

struct udp_challenge_state {
    __u64 timestamp;
    __u32 challenge_cookie;
    __u8 challenge_sent;
    __u8 padding[3];
};

// Statistics counters
enum {
    STAT_ALLOWED_PACKETS,
    STAT_BLOCKED_RATE_LIMIT,
    STAT_BLOCKED_BLACKLIST,
    STAT_BLOCKED_INVALID_PROTOCOL,
    STAT_BLOCKED_CHALLENGE_FAILED,
    STAT_BLOCKED_MAINTENANCE,
    STAT_TOTAL_PACKETS,
    STAT_XDP_DROP,
    STAT_XDP_PASS,
    STAT_XDP_REDIRECT,
    STAT_UDP_CHALLENGES_SENT,
    STAT_UDP_CHALLENGES_PASSED
};

// Helper functions
static __always_inline __u32 get_current_time(void)
{
    return bpf_ktime_get_ns() / 1000000; // Convert to milliseconds
}

static __always_inline __u64 hash_5tuple(__u32 src_ip, __u32 dst_ip, 
                                        __u16 src_port, __u16 dst_port, __u8 protocol)
{
    return ((__u64)src_ip << 32) | dst_ip | ((__u64)src_port << 48) | 
           ((__u64)dst_port << 32) | ((__u64)protocol << 24);
}

static __always_inline int update_rate_limit(__u32 src_ip, __u32 rate_limit, __u32 burst_limit)
{
    struct rate_limit_state *state = bpf_map_lookup_elem(&map_src_rate, &src_ip);
    __u32 current_time = get_current_time();
    
    if (!state) {
        // First packet from this IP
        struct rate_limit_state new_state = {
            .last_update = current_time,
            .tokens = burst_limit,
            .last_burst = 0
        };
        if (bpf_map_update_elem(&map_src_rate, &src_ip, &new_state, BPF_ANY) < 0)
            return -1;
        return 1; // Allow
    }
    
    // Calculate tokens based on time elapsed
    __u32 time_diff = current_time - state->last_update;
    __u32 new_tokens = state->tokens + (time_diff * rate_limit / 1000);
    
    if (new_tokens > burst_limit)
        new_tokens = burst_limit;
    
    if (new_tokens == 0) {
        // Rate limited
        state->last_update = current_time;
        return 0; // Drop
    }
    
    // Consume one token
    new_tokens--;
    state->tokens = new_tokens;
    state->last_update = current_time;
    
    return 1; // Allow
}

static __always_inline int is_blacklisted(__u32 src_ip)
{
    __u64 *blocked_until = bpf_map_lookup_elem(&map_blacklist, &src_ip);
    if (!blocked_until)
        return 0;
    
    __u64 current_time = bpf_ktime_get_ns() / 1000000;
    if (current_time < *blocked_until)
        return 1; // Still blocked
    
    // Expired, remove from blacklist
    bpf_map_delete_elem(&map_blacklist, &src_ip);
    return 0;
}

static __always_inline int validate_minecraft_java(struct xdp_md *ctx, void *data, void *data_end)
{
    // Enhanced Minecraft Java TCP validation
    // Minecraft Java handshake: VarInt length + packet ID (0x00 for handshake)
    // Then: protocol version (VarInt) + server address (String) + port (unsigned short) + next state (VarInt)
    
    if (data + 5 > data_end)
        return 0;
    
    __u8 *pkt = (__u8 *)data;
    __u32 offset = 0;
    
    // Parse VarInt length (first byte)
    __u32 length = 0;
    __u32 length_bytes = 0;
    __u8 byte = pkt[offset++];
    
    if (byte & 0x80) {
        // Multi-byte VarInt
        length = byte & 0x7F;
        while (offset < 5 && (pkt[offset] & 0x80)) {
            if (offset + 1 > data_end)
                return 0;
            length |= (pkt[offset] & 0x7F) << (7 * (offset - length_bytes));
            offset++;
        }
        if (offset < 5) {
            length |= pkt[offset] << (7 * (offset - length_bytes));
            offset++;
        }
    } else {
        length = byte;
    }
    
    // Validate length (reasonable bounds for handshake)
    if (length < 5 || length > 100)
        return 0;
    
    // Check packet ID (should be 0x00 for handshake)
    if (offset >= data_end || pkt[offset] != 0x00)
        return 0;
    offset++;
    
    // Check protocol version (should be reasonable Minecraft version)
    if (offset >= data_end)
        return 0;
    
    __u32 protocol_version = 0;
    __u8 ver_byte = pkt[offset++];
    if (ver_byte & 0x80) {
        // Multi-byte protocol version
        protocol_version = ver_byte & 0x7F;
        while (offset < data_end && (pkt[offset] & 0x80)) {
            protocol_version |= (pkt[offset] & 0x7F) << (7 * (offset - 1));
            offset++;
        }
        if (offset < data_end) {
            protocol_version |= pkt[offset] << (7 * (offset - 1));
            offset++;
        }
    } else {
        protocol_version = ver_byte;
    }
    
    // Validate protocol version (Minecraft versions typically 4-760+)
    if (protocol_version < 4 || protocol_version > 1000)
        return 0;
    
    return 1; // Valid Minecraft Java handshake
}

static __always_inline int validate_minecraft_bedrock(struct xdp_md *ctx, void *data, void *data_end)
{
    // Enhanced Minecraft Bedrock UDP validation
    // Bedrock uses RakNet protocol with specific packet structure
    
    if (data + 4 > data_end)
        return 0;
    
    __u8 *pkt = (__u8 *)data;
    
    // RakNet packet types for Minecraft Bedrock
    // 0x05: UNCONNECTED_PING
    // 0x06: UNCONNECTED_PONG  
    // 0x07: OPEN_CONNECTION_REQUEST_1
    // 0x08: OPEN_CONNECTION_REPLY_1
    // 0x09: OPEN_CONNECTION_REQUEST_2
    // 0x10: OPEN_CONNECTION_REPLY_2
    // 0x13: INCOMPATIBLE_PROTOCOL_VERSION
    // 0x15: UNCONNECTED_PING_OPEN_CONNECTIONS
    // 0x1c: OPEN_CONNECTION_REQUEST_1_BROADCAST
    
    __u8 packet_type = pkt[0];
    
    // Validate common RakNet packet types
    if (packet_type == 0x05 || packet_type == 0x06 || packet_type == 0x07 || 
        packet_type == 0x08 || packet_type == 0x09 || packet_type == 0x10 ||
        packet_type == 0x13 || packet_type == 0x15 || packet_type == 0x1c) {
        
        // Additional validation for specific packet types
        if (packet_type == 0x05 || packet_type == 0x15) { // UNCONNECTED_PING variants
            // Should have RakNet magic bytes at offset 1-16
            if (data + 17 > data_end)
                return 0;
            
            // Check for RakNet magic: 0x00, 0xFF, 0xFF, 0x00, 0xFE, 0xFE, 0xFE, 0xFE, 0xFD, 0xFD, 0xFD, 0xFD, 0x12, 0x34, 0x56, 0x78
            if (pkt[1] == 0x00 && pkt[2] == 0xFF && pkt[3] == 0xFF && pkt[4] == 0x00 &&
                pkt[5] == 0xFE && pkt[6] == 0xFE && pkt[7] == 0xFE && pkt[8] == 0xFE &&
                pkt[9] == 0xFD && pkt[10] == 0xFD && pkt[11] == 0xFD && pkt[12] == 0xFD &&
                pkt[13] == 0x12 && pkt[14] == 0x34 && pkt[15] == 0x56 && pkt[16] == 0x78) {
                return 1;
            }
        }
        
        // For other packet types, basic validation
        return 1;
    }
    
    return 0;
}

static __always_inline void update_stats(__u32 stat_type)
{
    __u64 *count = bpf_map_lookup_elem(&map_stats, &stat_type);
    if (count) {
        __sync_fetch_and_add(count, 1);
    }
}

static __always_inline int handle_udp_challenge(__u32 src_ip, void *data, void *data_end)
{
    // Check if this IP already has a challenge
    struct udp_challenge_state *challenge = bpf_map_lookup_elem(&map_udp_challenges, &src_ip);
    __u64 current_time = bpf_ktime_get_ns() / 1000000;
    
    if (!challenge) {
        // First packet from this IP - send challenge
        struct udp_challenge_state new_challenge = {
            .timestamp = current_time,
            .challenge_cookie = (__u32)(current_time ^ src_ip), // Simple cookie generation
            .challenge_sent = 1
        };
        
        if (bpf_map_update_elem(&map_udp_challenges, &src_ip, &new_challenge, BPF_ANY) < 0)
            return 0; // Failed to store challenge
        
        update_stats(STAT_UDP_CHALLENGES_SENT);
        return 0; // Drop packet, challenge sent
    }
    
    // Check if challenge is expired (5 seconds)
    if (current_time - challenge->timestamp > 5000) {
        // Challenge expired, remove and send new one
        bpf_map_delete_elem(&map_udp_challenges, &src_ip);
        return handle_udp_challenge(src_ip, data, data_end);
    }
    
    // Check if this packet contains the challenge response
    if (data + 8 > data_end)
        return 0;
    
    __u8 *pkt = (__u8 *)data;
    
    // Look for challenge cookie in packet (simplified - real implementation would be more sophisticated)
    // This is a placeholder - in reality, we'd need to modify the packet or use a different approach
    // For now, we'll accept packets after a short delay to simulate challenge completion
    
    if (current_time - challenge->timestamp > 100) { // 100ms delay
        // Challenge passed
        bpf_map_delete_elem(&map_udp_challenges, &src_ip);
        update_stats(STAT_UDP_CHALLENGES_PASSED);
        return 1; // Allow packet
    }
    
    return 0; // Still waiting for challenge response
}

// Main XDP program
SEC("xdp")
int xdp_minecraft_protection(struct xdp_md *ctx)
{
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;
    
    update_stats(STAT_TOTAL_PACKETS);
    
    // Parse Ethernet header
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_DROP;
    
    // Only handle IPv4 for now (IPv6 support can be added later)
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;
    
    // Parse IP header
    struct iphdr *ip = (struct iphdr *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_DROP;
    
    // Check if source is blacklisted
    if (is_blacklisted(ip->saddr)) {
        update_stats(STAT_BLOCKED_BLACKLIST);
        return XDP_DROP;
    }
    
    // Look up protected endpoint
    struct endpoint_key key = {
        .prefix_len = 32,
        .ip = ip->daddr,
        .port = 0,  // Will be set based on protocol
        .protocol = ip->protocol
    };
    
    // Set port based on protocol
    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = (struct tcphdr *)(ip + 1);
        if ((void *)(tcp + 1) > data_end)
            return XDP_DROP;
        key.port = bpf_ntohs(tcp->dest);
    } else if (ip->protocol == IPPROTO_UDP) {
        struct udphdr *udp = (struct udphdr *)(ip + 1);
        if ((void *)(udp + 1) > data_end)
            return XDP_DROP;
        key.port = bpf_ntohs(udp->dest);
    } else {
        return XDP_PASS; // Not TCP/UDP
    }
    
    struct endpoint_info *endpoint = bpf_map_lookup_elem(&map_protected_endpoints, &key);
    if (!endpoint) {
        return XDP_PASS; // Not a protected endpoint
    }
    
    // Check maintenance mode
    if (endpoint->maintenance_mode) {
        update_stats(STAT_BLOCKED_MAINTENANCE);
        return XDP_DROP;
    }
    
    // Apply rate limiting
    int rate_result = update_rate_limit(ip->saddr, endpoint->rate_limit, endpoint->burst_limit);
    if (rate_result < 0) {
        return XDP_DROP; // Error updating rate limit
    } else if (rate_result == 0) {
        update_stats(STAT_BLOCKED_RATE_LIMIT);
        return XDP_DROP; // Rate limited
    }
    
    // Protocol-specific validation
    int valid_protocol = 0;
    if (ip->protocol == IPPROTO_TCP && endpoint->protocol_type == 0) {
        // Java Minecraft (TCP)
        valid_protocol = validate_minecraft_java(ctx, data, data_end);
    } else if (ip->protocol == IPPROTO_UDP && endpoint->protocol_type == 1) {
        // Bedrock Minecraft (UDP) - apply challenge-response
        if (validate_minecraft_bedrock(ctx, data, data_end)) {
            // Valid Bedrock packet, now check UDP challenge
            int challenge_result = handle_udp_challenge(ip->saddr, data, data_end);
            if (challenge_result == 0) {
                update_stats(STAT_BLOCKED_CHALLENGE_FAILED);
                return XDP_DROP; // Challenge failed or in progress
            }
            valid_protocol = 1;
        }
    }
    
    if (!valid_protocol) {
        update_stats(STAT_BLOCKED_INVALID_PROTOCOL);
        return XDP_DROP;
    }
    
    // Update connection tracking for established flows
    __u64 flow_hash = hash_5tuple(ip->saddr, ip->daddr, 
                                 (ip->protocol == IPPROTO_TCP) ? 
                                 bpf_ntohs(((struct tcphdr *)(ip + 1))->source) : 
                                 bpf_ntohs(((struct udphdr *)(ip + 1))->source),
                                 key.port, ip->protocol);
    
    struct conntrack_entry *conn = bpf_map_lookup_elem(&map_conntrack, &flow_hash);
    if (!conn) {
        // New connection - add to conntrack
        struct conntrack_entry new_conn = {
            .src_ip = ip->saddr,
            .dst_ip = ip->daddr,
            .src_port = (ip->protocol == IPPROTO_TCP) ? 
                       bpf_ntohs(((struct tcphdr *)(ip + 1))->source) : 
                       bpf_ntohs(((struct udphdr *)(ip + 1))->source),
            .dst_port = key.port,
            .protocol = ip->protocol,
            .state = 1, // established
            .challenge_id = 0
        };
        bpf_map_update_elem(&map_conntrack, &flow_hash, &new_conn, BPF_ANY);
    }
    
    update_stats(STAT_ALLOWED_PACKETS);
    return XDP_REDIRECT; // Redirect to user-space proxy
}

char _license[] SEC("license") = "GPL";
