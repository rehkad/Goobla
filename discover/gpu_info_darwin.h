#import <Metal/Metal.h>
#include <stdint.h>
uint64_t getRecommendedMaxVRAM();
uint64_t getPhysicalMemory();
uint64_t getFreeMemory();
// getCurrentAllocatedVRAM returns the number of bytes of VRAM currently
// allocated by the system's default Metal device. Returns 0 on platforms that
// do not support querying this value.
uint64_t getCurrentAllocatedVRAM();
