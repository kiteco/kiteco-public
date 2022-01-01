// +build darwin

package performance

/*
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework IOKit
#cgo CFLAGS: -x objective-c
#include <string.h>
#include <stdio.h>
#include "include/kite_smc.c"

#define FAN_0            "F0Ac"
#define FAN_0_MIN_RPM    "F0Mn"
#define FAN_0_MAX_RPM    "F0Mx"
#define FAN_0_SAFE_RPM   "F0Sf"
#define FAN_0_TARGET_RPM "F0Tg"
#define FAN_1            "F1Ac"
#define FAN_1_MIN_RPM    "F1Mn"
#define FAN_1_MAX_RPM    "F1Mx"
#define FAN_1_SAFE_RPM   "F1Sf"
#define FAN_1_TARGET_RPM "F1Tg"
#define FAN_2            "F2Ac"
#define FAN_2_MIN_RPM    "F2Mn"
#define FAN_2_MAX_RPM    "F2Mx"
#define FAN_2_SAFE_RPM   "F2Sf"
#define FAN_2_TARGET_RPM "F2Tg"
#define NUM_FANS         "FNum"
#define FORCE_BITS       "FS! "

#define DATA_TYPE_FPE2   "fpe2"
#define DATA_TYPE_UINT8  "ui8 "

static unsigned int kite_from_fpe2(uint8_t data[32])
{
    unsigned int ans = 0;

    // Data type for fan calls - fpe2
    // This is assumend to mean floating point, with 2 exponent bits
    // SO link: /questions/22160746/fpe2-and-sp78-data-types
    ans += data[0] << 6;
    ans += data[1] << 2;

    return ans;
}

unsigned int get_fan_rpm(unsigned int fan_num)
{
    char key[5];
    kern_return_t result;
    kite_smc_return_t  result_smc;

    sprintf(key, "F%dAc", fan_num);
    result = kite_read_smc(key, &result_smc);

    if (!(result == kIOReturnSuccess &&
          result_smc.data_size == 2 &&
          result_smc.data_type == kite_to_uint32(DATA_TYPE_FPE2))) {
        // Error
        return 0;
    }

    return kite_from_fpe2(result_smc.data);
}

int get_num_fans(void)
{
    kern_return_t result;
    kite_smc_return_t  result_smc;

    result = kite_read_smc(NUM_FANS, &result_smc);

    if (!(result == kIOReturnSuccess &&
          result_smc.data_size == 1   &&
          result_smc.data_type == kite_to_uint32(DATA_TYPE_UINT8))) {
        // Error
        return -1;
    }

    return result_smc.data[0];
}
*/
import "C"

import (
	"errors"
	"fmt"

	"github.com/shirou/gopsutil/load"
)

func fanSpeedsImpl() ([]FanSpeedStat, error) {
	C.kite_open_smc()
	defer C.kite_close_smc()

	numFans := C.uint(C.get_num_fans())
	if numFans == 0 {
		return nil, errors.New("No fan detected")
	}

	// only single-digit fans supported
	if numFans > 10 {
		numFans = C.uint(10)
	}

	result := make([]FanSpeedStat, 0, numFans)
	for i := C.uint(0); i < numFans; i++ {
		fanSpeed := C.get_fan_rpm(i)
		if fanSpeed > 0 {

			result = append(result, FanSpeedStat{
				SensorKey: fmt.Sprintf("Fan_%d", i),
				Speed:     float64(fanSpeed),
			})
		}
	}

	return result, nil
}

func loadAvgImpl() []float64 {
	result := make([]float64, 3)
	avg, err := load.Avg()
	if err != nil {
		return nil
	}
	result[0] = avg.Load1
	result[1] = avg.Load5
	result[2] = avg.Load15
	return result
}
