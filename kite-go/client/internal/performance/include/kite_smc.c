#include "kite_smc.h"

#define IOSERVICE_SMC "AppleSMC"
#define IOSERVICE_MODEL "IOPlatformExpertDevice"

#define DATA_TYPE_SP78 "sp78"

typedef enum {
  kite_kSMCUserClientOpen = 0,
  kite_kSMCUserClientClose = 1,
  kite_kSMCHandleYPCEvent = 2,
  kite_kSMCReadKey = 5,
  kite_kSMCWriteKey = 6,
  kite_kSMCGetKeyCount = 7,
  kite_kSMCGetKeyFromIndex = 8,
  kite_kSMCGetKeyInfo = 9,
} kite_selector_t;

typedef struct {
  unsigned char major;
  unsigned char minor;
  unsigned char build;
  unsigned char reserved;
  unsigned short release;
} kite_SMCVersion;

typedef struct {
  uint16_t version;
  uint16_t length;
  uint32_t cpuPLimit;
  uint32_t gpuPLimit;
  uint32_t memPLimit;
} kite_SMCPLimitData;

typedef struct {
  IOByteCount data_size;
  uint32_t data_type;
  uint8_t data_attributes;
} kite_SMCKeyInfoData;

typedef struct {
  uint32_t key;
  kite_SMCVersion vers;
  kite_SMCPLimitData p_limit_data;
  kite_SMCKeyInfoData key_info;
  uint8_t result;
  uint8_t status;
  uint8_t data8;
  uint32_t data32;
  uint8_t bytes[32];
} kite_SMCParamStruct;

typedef enum {
  kite_kSMCSuccess = 0,
  kite_kSMCError = 1,
  kite_kSMCKeyNotFound = 0x84,
} kite_kSMC_t;

typedef struct {
  uint8_t data[32];
  uint32_t data_type;
  uint32_t data_size;
  kite_kSMC_t kSMC;
} kite_smc_return_t;

static const int kite_SMC_KEY_SIZE = 4; // number of characters in an SMC key.
static io_connect_t kite_conn;          // our connection to the SMC.

kern_return_t kite_open_smc(void) {
  kern_return_t result;
  io_service_t service;

  service = IOServiceGetMatchingService(kIOMasterPortDefault,
                                        IOServiceMatching(IOSERVICE_SMC));
  if (service == 0) {
    // Note: IOServiceMatching documents 0 on failure
    printf("ERROR: %s NOT FOUND\n", IOSERVICE_SMC);
    return kIOReturnError;
  }

  result = IOServiceOpen(service, mach_task_self(), 0, &kite_conn);
  IOObjectRelease(service);

  return result;
}

kern_return_t kite_close_smc(void) { return IOServiceClose(kite_conn); }

static uint32_t kite_to_uint32(char *key) {
  uint32_t ans = 0;
  uint32_t shift = 24;

  if (strlen(key) != kite_SMC_KEY_SIZE) {
    return 0;
  }

  for (int i = 0; i < kite_SMC_KEY_SIZE; i++) {
    ans += key[i] << shift;
    shift -= 8;
  }

  return ans;
}

static kern_return_t kite_call_smc(kite_SMCParamStruct *input, kite_SMCParamStruct *output) {
  kern_return_t result;
  size_t input_cnt = sizeof(kite_SMCParamStruct);
  size_t output_cnt = sizeof(kite_SMCParamStruct);

  result = IOConnectCallStructMethod(kite_conn, kite_kSMCHandleYPCEvent, input, input_cnt,
                                     output, &output_cnt);

  if (result != kIOReturnSuccess) {
    result = err_get_code(result);
  }
  return result;
}

static kern_return_t kite_read_smc(char *key, kite_smc_return_t *result_smc) {
  kern_return_t result;
  kite_SMCParamStruct input;
  kite_SMCParamStruct output;

  memset(&input, 0, sizeof(kite_SMCParamStruct));
  memset(&output, 0, sizeof(kite_SMCParamStruct));
  memset(result_smc, 0, sizeof(kite_smc_return_t));

  input.key = kite_to_uint32(key);
  input.data8 = kite_kSMCGetKeyInfo;

  result = kite_call_smc(&input, &output);
  result_smc->kSMC = output.result;

  if (result != kIOReturnSuccess || output.result != kite_kSMCSuccess) {
    return result;
  }

  result_smc->data_size = output.key_info.data_size;
  result_smc->data_type = output.key_info.data_type;

  input.key_info.data_size = output.key_info.data_size;
  input.data8 = kite_kSMCReadKey;

  result = kite_call_smc(&input, &output);
  result_smc->kSMC = output.result;

  if (result != kIOReturnSuccess || output.result != kite_kSMCSuccess) {
    return result;
  }

  memcpy(result_smc->data, output.bytes, sizeof(output.bytes));

  return result;
}
