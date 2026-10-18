#ifndef WHALE_VOICE_CAPTURE_H
#define WHALE_VOICE_CAPTURE_H
#include <stddef.h>
typedef struct WhaleMicPermission WhaleMicPermission;
typedef struct WhaleMic WhaleMic;
WhaleMicPermission *whale_mic_permission(void);
int whale_mic_permission_status(WhaleMicPermission *p);
void whale_mic_permission_release(WhaleMicPermission *p);
WhaleMic *whale_mic_start(size_t limit, int *error);
int whale_mic_finished(WhaleMic *m);
int whale_mic_close(WhaleMic *m);
const void *whale_mic_bytes(WhaleMic *m, size_t *length);
void whale_mic_free(WhaleMic *m);
#endif
