#include <breez_liquid_sdk.h>

// This file exists beacause of
// https://github.com/golang/go/issues/11263

void cgo_rust_task_callback_bridge_breez_liquid_sdk(RustTaskCallback cb, const void * taskData, int8_t status) {
  cb(taskData, status);
}