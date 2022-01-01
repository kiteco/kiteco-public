// +build darwin

package benchmark

/*
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework IOKit
#cgo CFLAGS: -x objective-c

#include <sys/sysctl.h>
#include <utmpx.h>
#include <mach/mach.h>

#import <ApplicationServices/ApplicationServices.h>
#import <Foundation/Foundation.h>

static inline unsigned long menubarMemoryUse(void) {
  struct task_basic_info info;
  mach_msg_type_number_t size = sizeof(info);
  kern_return_t kerr = task_info(mach_task_self(),
                                 TASK_BASIC_INFO,
                                 (task_info_t)&info,
                                 &size);
  if (kerr == KERN_SUCCESS) {
    return info.resident_size;
  } else {
    return -1;
  }
}

// Warning: This code is from the internet:
// SO link: /questions/8223348/ios-get-cpu-usage-from-application
float cpuUsage() {
    kern_return_t kr;
    task_info_data_t tinfo;
    mach_msg_type_number_t task_info_count;

    task_info_count = TASK_INFO_MAX;
    kr = task_info(mach_task_self(), TASK_BASIC_INFO, (task_info_t)tinfo, &task_info_count);
    if (kr != KERN_SUCCESS) {
        return -1;
    }

    task_basic_info_t      basic_info;
    thread_array_t         thread_list;
    mach_msg_type_number_t thread_count;

    thread_info_data_t     thinfo;
    mach_msg_type_number_t thread_info_count;

    thread_basic_info_t basic_info_th;
    uint32_t stat_thread = 0; // Mach threads

    basic_info = (task_basic_info_t)tinfo;

    // get threads in the task
    kr = task_threads(mach_task_self(), &thread_list, &thread_count);
    if (kr != KERN_SUCCESS) {
        return -2;
    }
    if (thread_count > 0) {
        stat_thread += thread_count;
    }

    long tot_sec = 0;
    long tot_usec = 0;
    float tot_cpu = 0;
    int j;

    for (j = 0; j < (int)thread_count; j++) {
        thread_info_count = THREAD_INFO_MAX;
        kr = thread_info(thread_list[j], THREAD_BASIC_INFO,
                         (thread_info_t)thinfo, &thread_info_count);

        if (kr != KERN_SUCCESS) {
            return -3;
        }

        basic_info_th = (thread_basic_info_t)thinfo;

        if (!(basic_info_th->flags & TH_FLAGS_IDLE)) {
            tot_sec = tot_sec + basic_info_th->user_time.seconds + basic_info_th->system_time.seconds;
            tot_usec = tot_usec + basic_info_th->user_time.microseconds + basic_info_th->system_time.microseconds;
            tot_cpu = tot_cpu + basic_info_th->cpu_usage / (float)TH_USAGE_SCALE * 100.0;
        }

    } // for each thread

    kr = vm_deallocate(mach_task_self(), (vm_offset_t)thread_list, thread_count * sizeof(thread_t));
    if (kr != KERN_SUCCESS) {
        return -4;
    }

    return tot_cpu;
}

*/
import "C"

// cpuUsage gets the current CPU usage
func cpuUsage() float64 {
	return float64(C.cpuUsage())
}
