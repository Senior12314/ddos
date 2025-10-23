/*
 * CloudNordSP XDP Loader and Map Manager
 * 
 * This program loads the XDP eBPF program and provides utilities
 * for managing BPF maps from user space.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <sys/resource.h>
#include <bpf/bpf.h>
#include <bpf/libbpf.h>
#include <net/if.h>
#include <linux/if_link.h>

// Map file descriptors
static int map_protected_endpoints_fd;
static int map_src_rate_fd;
static int map_conntrack_fd;
static int map_blacklist_fd;
static int map_stats_fd;
static int map_udp_challenges_fd;

// XDP program object
static struct bpf_object *obj;

// Load XDP program
static int load_xdp_program(const char *ifname, const char *filename)
{
    int err, prog_fd;
    struct bpf_program *prog;
    struct bpf_link *link;
    
    // Set resource limits for eBPF
    struct rlimit r = {RLIM_INFINITY, RLIM_INFINITY};
    setrlimit(RLIMIT_MEMLOCK, &r);
    
    // Load eBPF object
    obj = bpf_object__open_file(filename, NULL);
    if (libbpf_get_error(obj)) {
        fprintf(stderr, "Failed to open eBPF object: %s\n", 
                libbpf_strerror(libbpf_get_error(obj)));
        return -1;
    }
    
    // Load eBPF program
    err = bpf_object__load(obj);
    if (err) {
        fprintf(stderr, "Failed to load eBPF object: %s\n", 
                libbpf_strerror(err));
        return -1;
    }
    
    // Find the XDP program
    prog = bpf_object__find_program_by_name(obj, "xdp_minecraft_protection");
    if (!prog) {
        fprintf(stderr, "Failed to find XDP program\n");
        return -1;
    }
    
    // Get program file descriptor
    prog_fd = bpf_program__fd(prog);
    if (prog_fd < 0) {
        fprintf(stderr, "Failed to get program FD: %s\n", 
                libbpf_strerror(prog_fd));
        return -1;
    }
    
    // Attach XDP program to interface
    int ifindex = if_nametoindex(ifname);
    if (ifindex == 0) {
        fprintf(stderr, "Failed to get interface index for %s\n", ifname);
        return -1;
    }
    
    link = bpf_program__attach_xdp(prog, ifindex);
    if (libbpf_get_error(link)) {
        fprintf(stderr, "Failed to attach XDP program: %s\n", 
                libbpf_strerror(libbpf_get_error(link)));
        return -1;
    }
    
    printf("XDP program attached to interface %s\n", ifname);
    
    // Get map file descriptors
    map_protected_endpoints_fd = bpf_object__find_map_fd_by_name(obj, "map_protected_endpoints");
    map_src_rate_fd = bpf_object__find_map_fd_by_name(obj, "map_src_rate");
    map_conntrack_fd = bpf_object__find_map_fd_by_name(obj, "map_conntrack");
    map_blacklist_fd = bpf_object__find_map_fd_by_name(obj, "map_blacklist");
    map_stats_fd = bpf_object__find_map_fd_by_name(obj, "map_stats");
    map_udp_challenges_fd = bpf_object__find_map_fd_by_name(obj, "map_udp_challenges");
    
    if (map_protected_endpoints_fd < 0 || map_src_rate_fd < 0 || 
        map_conntrack_fd < 0 || map_blacklist_fd < 0 || map_stats_fd < 0 ||
        map_udp_challenges_fd < 0) {
        fprintf(stderr, "Failed to get map file descriptors\n");
        return -1;
    }
    
    return 0;
}

// Add protected endpoint
int add_protected_endpoint(__u32 front_ip, __u16 front_port, __u8 protocol,
                          __u32 origin_ip, __u16 origin_port, __u8 protocol_type,
                          __u32 rate_limit, __u32 burst_limit)
{
    struct endpoint_key key = {
        .prefix_len = 32,
        .ip = front_ip,
        .port = front_port,
        .protocol = protocol
    };
    
    struct endpoint_info info = {
        .origin_ip = origin_ip,
        .origin_port = origin_port,
        .rate_limit = rate_limit,
        .burst_limit = burst_limit,
        .protocol_type = protocol_type,
        .maintenance_mode = 0,
        .padding = {0, 0}
    };
    
    int err = bpf_map_update_elem(map_protected_endpoints_fd, &key, &info, BPF_ANY);
    if (err) {
        fprintf(stderr, "Failed to add protected endpoint: %s\n", strerror(errno));
        return -1;
    }
    
    printf("Added protected endpoint: %u.%u.%u.%u:%u -> %u.%u.%u.%u:%u\n",
           (front_ip >> 24) & 0xFF, (front_ip >> 16) & 0xFF, 
           (front_ip >> 8) & 0xFF, front_ip & 0xFF, front_port,
           (origin_ip >> 24) & 0xFF, (origin_ip >> 16) & 0xFF, 
           (origin_ip >> 8) & 0xFF, origin_ip & 0xFF, origin_port);
    
    return 0;
}

// Remove protected endpoint
int remove_protected_endpoint(__u32 front_ip, __u16 front_port, __u8 protocol)
{
    struct endpoint_key key = {
        .prefix_len = 32,
        .ip = front_ip,
        .port = front_port,
        .protocol = protocol
    };
    
    int err = bpf_map_delete_elem(map_protected_endpoints_fd, &key);
    if (err) {
        fprintf(stderr, "Failed to remove protected endpoint: %s\n", strerror(errno));
        return -1;
    }
    
    printf("Removed protected endpoint: %u.%u.%u.%u:%u\n",
           (front_ip >> 24) & 0xFF, (front_ip >> 16) & 0xFF, 
           (front_ip >> 8) & 0xFF, front_ip & 0xFF, front_port);
    
    return 0;
}

// Add IP to blacklist
int add_to_blacklist(__u32 ip, __u64 duration_ms)
{
    __u64 block_until = (__u64)time(NULL) * 1000 + duration_ms;
    
    int err = bpf_map_update_elem(map_blacklist_fd, &ip, &block_until, BPF_ANY);
    if (err) {
        fprintf(stderr, "Failed to add IP to blacklist: %s\n", strerror(errno));
        return -1;
    }
    
    printf("Added IP to blacklist: %u.%u.%u.%u (until %llu)\n",
           (ip >> 24) & 0xFF, (ip >> 16) & 0xFF, 
           (ip >> 8) & 0xFF, ip & 0xFF, block_until);
    
    return 0;
}

// Remove IP from blacklist
int remove_from_blacklist(__u32 ip)
{
    int err = bpf_map_delete_elem(map_blacklist_fd, &ip);
    if (err) {
        fprintf(stderr, "Failed to remove IP from blacklist: %s\n", strerror(errno));
        return -1;
    }
    
    printf("Removed IP from blacklist: %u.%u.%u.%u\n",
           (ip >> 24) & 0xFF, (ip >> 16) & 0xFF, 
           (ip >> 8) & 0xFF, ip & 0xFF);
    
    return 0;
}

// Get statistics
int get_stats(__u64 *stats, size_t count)
{
    for (size_t i = 0; i < count; i++) {
        __u32 key = i;
        __u64 *value = bpf_map_lookup_elem(map_stats_fd, &key);
        if (value) {
            stats[i] = *value;
        } else {
            stats[i] = 0;
        }
    }
    return 0;
}

// Print statistics
void print_stats(void)
{
    __u64 stats[12];
    get_stats(stats, 12);
    
    printf("\n=== CloudNordSP Statistics ===\n");
    printf("Total packets processed: %llu\n", stats[6]);
    printf("Allowed packets: %llu\n", stats[0]);
    printf("Blocked - Rate limit: %llu\n", stats[1]);
    printf("Blocked - Blacklist: %llu\n", stats[2]);
    printf("Blocked - Invalid protocol: %llu\n", stats[3]);
    printf("Blocked - Challenge failed: %llu\n", stats[4]);
    printf("Blocked - Maintenance: %llu\n", stats[5]);
    printf("XDP drops: %llu\n", stats[7]);
    printf("XDP passes: %llu\n", stats[8]);
    printf("XDP redirects: %llu\n", stats[9]);
    printf("UDP challenges sent: %llu\n", stats[10]);
    printf("UDP challenges passed: %llu\n", stats[11]);
    printf("==============================\n");
}

// Cleanup
void cleanup(void)
{
    if (obj) {
        bpf_object__close(obj);
    }
}

// Main function for CLI usage
int main(int argc, char *argv[])
{
    if (argc < 3) {
        printf("Usage: %s <interface> <command> [args...]\n", argv[0]);
        printf("Commands:\n");
        printf("  load <xdp_file>                    - Load XDP program\n");
        printf("  add-endpoint <front_ip> <front_port> <protocol> <origin_ip> <origin_port> <type> <rate> <burst>\n");
        printf("  remove-endpoint <front_ip> <front_port> <protocol>\n");
        printf("  blacklist <ip> <duration_ms>\n");
        printf("  unblacklist <ip>\n");
        printf("  stats\n");
        return 1;
    }
    
    const char *ifname = argv[1];
    const char *command = argv[2];
    
    if (strcmp(command, "load") == 0) {
        if (argc < 4) {
            printf("Usage: %s <interface> load <xdp_file>\n", argv[0]);
            return 1;
        }
        
        if (load_xdp_program(ifname, argv[3]) < 0) {
            return 1;
        }
        
        // Keep running to maintain the program
        printf("XDP program loaded. Press Ctrl+C to stop.\n");
        while (1) {
            sleep(1);
        }
    }
    
    printf("Unknown command: %s\n", command);
    return 1;
}
