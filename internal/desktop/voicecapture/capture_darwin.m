//go:build darwin && cgo

#import <AVFoundation/AVFoundation.h>
#import <AudioToolbox/AudioToolbox.h>
#import <Foundation/Foundation.h>
#include <stdatomic.h>
#include <stdbool.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>
#include "capture_darwin.h"

struct WhaleMicPermission { atomic_int references; atomic_int status; };
void whale_mic_permission_release(WhaleMicPermission *p) {
 if (p && atomic_fetch_sub(&p->references, 1) == 1) free(p);
}
WhaleMicPermission *whale_mic_permission(void) {
 WhaleMicPermission *p = calloc(1, sizeof(*p));
 if (!p) return NULL;
 atomic_init(&p->references, 1); atomic_init(&p->status, 0);
 @autoreleasepool {
  NSString *reason = [[NSBundle mainBundle] objectForInfoDictionaryKey:@"NSMicrophoneUsageDescription"];
  if (![reason isKindOfClass:[NSString class]] || reason.length == 0) { atomic_store(&p->status, -2); return p; }
  AVAuthorizationStatus status = [AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeAudio];
  if (status == AVAuthorizationStatusAuthorized) atomic_store(&p->status, 1);
  else if (status != AVAuthorizationStatusNotDetermined) atomic_store(&p->status, -1);
  else {
   // The callback owns a reference even if Go cancels while the OS prompt is open.
   atomic_fetch_add(&p->references, 1);
   [AVCaptureDevice requestAccessForMediaType:AVMediaTypeAudio completionHandler:^(BOOL granted) {
    atomic_store(&p->status, granted ? 1 : -1);
    whale_mic_permission_release(p);
   }];
  }
 }
 return p;
}
int whale_mic_permission_status(WhaleMicPermission *p) { return atomic_load(&p->status); }

struct WhaleMic {
 AudioQueueRef queue;
 pthread_mutex_t lock;
 unsigned char *bytes;
 size_t limit, length;
 int stopping, error;
};
static void whale_mic_input(void *opaque, AudioQueueRef queue, AudioQueueBufferRef buffer,
 const AudioTimeStamp *stamp, UInt32 packets, const AudioStreamPacketDescription *descriptions) {
 (void)stamp; (void)packets; (void)descriptions;
 WhaleMic *m = opaque;
 pthread_mutex_lock(&m->lock);
 if (!m->stopping && !m->error) {
  size_t n = buffer->mAudioDataByteSize;
  if (n > m->limit - m->length) n = m->limit - m->length;
  n -= n % 2;
  memcpy(m->bytes + m->length, buffer->mAudioData, n); m->length += n;
  if (m->length < m->limit) {
   OSStatus status = AudioQueueEnqueueBuffer(queue, buffer, 0, NULL);
   if (status) m->error = (int)status;
  }
 }
 pthread_mutex_unlock(&m->lock);
}
WhaleMic *whale_mic_start(size_t limit, int *error) {
 *error = 0;
 WhaleMic *m = calloc(1, sizeof(*m));
 if (!m) { *error = -1; return NULL; }
 pthread_mutex_init(&m->lock, NULL);
 m->limit = limit; m->bytes = calloc(1, limit);
 if (!m->bytes) { *error = -1; whale_mic_free(m); return NULL; }
 AudioStreamBasicDescription format = {0};
 format.mSampleRate = 16000; format.mFormatID = kAudioFormatLinearPCM;
 format.mFormatFlags = kLinearPCMFormatFlagIsSignedInteger | kLinearPCMFormatFlagIsPacked;
 format.mBytesPerPacket = 2; format.mFramesPerPacket = 1; format.mBytesPerFrame = 2;
 format.mChannelsPerFrame = 1; format.mBitsPerChannel = 16;
 OSStatus status = AudioQueueNewInput(&format, whale_mic_input, m, NULL, NULL, 0, &m->queue);
 for (int i = 0; !status && i < 3; i++) {
  AudioQueueBufferRef buffer = NULL;
  status = AudioQueueAllocateBuffer(m->queue, 3200, &buffer);
  if (!status) status = AudioQueueEnqueueBuffer(m->queue, buffer, 0, NULL);
 }
 if (!status) status = AudioQueueStart(m->queue, NULL);
 if (status) {
  *error = (int)status;
  // On disposal failure quarantine the native allocation, never free callback state.
  if (whale_mic_close(m) == 0) whale_mic_free(m); else *error = -2;
  return NULL;
 }
 return m;
}
int whale_mic_finished(WhaleMic *m) {
 pthread_mutex_lock(&m->lock);
 int done = m->error || m->length >= m->limit;
 pthread_mutex_unlock(&m->lock); return done;
}
int whale_mic_close(WhaleMic *m) {
 pthread_mutex_lock(&m->lock); m->stopping = 1; pthread_mutex_unlock(&m->lock);
 if (m->queue) {
  // Immediate synchronous disposal waits for callbacks and frees queue buffers.
  OSStatus status = AudioQueueDispose(m->queue, true);
  if (status) {
   pthread_mutex_lock(&m->lock);
   memset(m->bytes, 0, m->limit); m->length = 0;
   pthread_mutex_unlock(&m->lock);
   return (int)status;
  }
  m->queue = NULL;
 }
 return 0;
}
const void *whale_mic_bytes(WhaleMic *m, size_t *length) {
 // Called only after successful synchronous disposal.
 *length = m->error ? 0 : m->length;
 return m->bytes;
}
void whale_mic_free(WhaleMic *m) {
 if (!m) return;
 if (m->bytes) { memset(m->bytes, 0, m->limit); free(m->bytes); }
 pthread_mutex_destroy(&m->lock); free(m);
}
