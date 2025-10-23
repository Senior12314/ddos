# CloudNordSP XDP eBPF Program Makefile

CLANG ?= clang
LLC ?= llc
BPFTOOL ?= bpftool

# Compiler flags
CLANG_FLAGS = -O2 -g -Wall -Wno-unused-value -Wno-pointer-sign \
              -Wno-compare-distinct-pointer-types \
              -Werror -emit-llvm -c

# Target files
TARGET = minecraft_protection
XDP_OBJ = $(TARGET).o
XDP_SRC = $(TARGET).c

# Default target
all: $(XDP_OBJ)

# Compile XDP program
$(XDP_OBJ): $(XDP_SRC)
	$(CLANG) $(CLANG_FLAGS) -target bpf -o $(XDP_OBJ) $(XDP_SRC)

# Load XDP program (requires root)
load: $(XDP_OBJ)
	sudo $(BPFTOOL) prog load $(XDP_OBJ) /sys/fs/bpf/$(TARGET)
	sudo $(BPFTOOL) net attach xdp id $$($(BPFTOOL) prog show name xdp_minecraft_protection | grep -o '[0-9]*' | head -1) dev lo

# Unload XDP program
unload:
	sudo $(BPFTOOL) net detach xdp dev lo
	sudo rm -f /sys/fs/bpf/$(TARGET)

# Show loaded programs
show:
	$(BPFTOOL) prog show name xdp_minecraft_protection

# Show maps
show-maps:
	$(BPFTOOL) map show

# Clean build artifacts
clean:
	rm -f $(XDP_OBJ)

# Install dependencies (Ubuntu/Debian)
install-deps:
	sudo apt-get update
	sudo apt-get install -y clang llvm libbpf-dev linux-headers-$(shell uname -r) bpftool

# Test with sample packets
test: load
	# This would typically involve sending test packets
	# For now, just verify the program loads successfully
	@echo "XDP program loaded successfully"
	@echo "Use 'make show' to see loaded programs"
	@echo "Use 'make unload' to remove the program"

.PHONY: all load unload show show-maps clean install-deps test
