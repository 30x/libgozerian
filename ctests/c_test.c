#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <CUnit/Basic.h>
#include <libgozerian.h>
#include "ctests.h"

#define HDR_BUF_SIZE 1024

unsigned int id;
unsigned int rid;

static char hdrBuf[HDR_BUF_SIZE];

static void initRequest(void) {
  id = GoCreateRequest(TEST_HANDLER);
  CU_ASSERT_NOT_EQUAL(id, 0);
  rid = GoCreateResponse(TEST_HANDLER);
  CU_ASSERT_NOT_EQUAL(rid, 0);
}

static void cleanRequest(void) {
  GoFreeRequest(id);
  GoFreeResponse(rid);
}

static void createHeader(const char* verb, const char* uri, int bodyLen, const char* cType) {
  char tmp[128];

  sprintf(hdrBuf, "%s %s HTTP/1.1\r\nHost: localhost:1234\r\n", verb, uri);
  if (bodyLen > 0) {
    sprintf(tmp, "Content-Length: %i\r\n", bodyLen);
    strcat(hdrBuf, tmp);
  }
  if (cType != NULL) {
    sprintf(tmp, "Content-Type: %s\r\n", cType);
    strcat(hdrBuf, tmp);
  }
  strcat(hdrBuf, "\r\n");
}

static void createResponse(int bodyLen, const char* cType) {
  char tmp[128];

  sprintf(hdrBuf, "Server: libgozerian\n");
  if (bodyLen > 0) {
    sprintf(tmp, "Content-Length: %i\r\n", bodyLen);
    strcat(hdrBuf, tmp);
  }
  if (cType != NULL) {
    sprintf(tmp, "Content-Type: %s\r\n", cType);
    strcat(hdrBuf, tmp);
  }
  strcat(hdrBuf, "\n");
}

static void test_bad_handler(void) {
  char* err = GoCreateHandler("bad", "urn:weaver-proxy:always-bad");
  CU_ASSERT_PTR_NOT_NULL(err);
  if (err != NULL) {
    free(err);
  }
}

static void test_basic_request(void) {
  initRequest();
  createHeader("GET", "/pass", 0, NULL);
  GoBeginRequest(id, hdrBuf);
  char* cmd = GoPollRequest(id, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "DONE");
  free(cmd);

  createResponse(10, "text/plain");
  GoBeginResponse(rid, id, 200, hdrBuf);
  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "DONE");
  free(cmd);

  cleanRequest();
}

static void test_replace_request_body(void) {
  initRequest();

  createHeader("POST", "/replacebody", 100, "text/plain");
  GoBeginRequest(id, hdrBuf);
  char* cmd = GoPollRequest(id, 1);
  CU_ASSERT_TRUE(strncmp("WBOD", cmd, 4) == 0);
  unsigned int chunkID = strtoul(cmd + 4, NULL, 16);
  free(cmd);
  char* chunk = (char*)GoGetChunk(chunkID);
  CU_ASSERT_PTR_NOT_NULL(chunk);
  unsigned int chunkLen = GoGetChunkLength(chunkID);
  CU_ASSERT_NOT_EQUAL(chunkLen, 0);
  CU_ASSERT_TRUE(strncmp("Hello! I am the server!", chunk, chunkLen) == 0);
  free(chunk);
  GoReleaseChunk(chunkID);

  cmd = GoPollRequest(id, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "DONE");
  free(cmd);

  cleanRequest();
}

static void test_replace_response_body(void) {
  initRequest();
  createHeader("GET", "/transformbody", 0, NULL);
  GoBeginRequest(id, hdrBuf);
  char* cmd = GoPollRequest(id, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "DONE");
  free(cmd);

  createResponse(10, "text/plain");
  GoBeginResponse(rid, id, 200, hdrBuf);
  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_TRUE(strncmp("WBOD", cmd, 4) == 0);
  unsigned int chunkID = strtoul(cmd + 4, NULL, 16);
  free(cmd);
  char* chunk = (char*)GoGetChunk(chunkID);
  CU_ASSERT_PTR_NOT_NULL(chunk);
  unsigned int chunkLen = GoGetChunkLength(chunkID);
  CU_ASSERT_NOT_EQUAL(chunkLen, 0);
  CU_ASSERT_TRUE(strncmp("We have transformed the response!", chunk, chunkLen) == 0);
  free(chunk);
  GoReleaseChunk(chunkID);

  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "DONE");
  free(cmd);

  cleanRequest();
}

static void test_replace_response_body_chunks(void) {
  initRequest();
  createHeader("GET", "/transformbodychunks", 0, NULL);
  GoBeginRequest(id, hdrBuf);
  char* cmd = GoPollRequest(id, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "DONE");
  free(cmd);

  createResponse(10, "text/plain");
  GoBeginResponse(rid, id, 200, hdrBuf);
  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_TRUE(strncmp("WHDR", cmd, 4) == 0);
  free(cmd);

  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "RBOD");
  free(cmd);
  char* bod = "Hello, Server!";
  GoSendResponseBodyChunk(rid, 1, bod, strlen(bod));

  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_TRUE(strncmp("WBOD", cmd, 4) == 0);
  unsigned int chunkID = strtoul(cmd + 4, NULL, 16);
  free(cmd);
  char* chunk = (char*)GoGetChunk(chunkID);
  CU_ASSERT_PTR_NOT_NULL(chunk);
  unsigned int chunkLen = GoGetChunkLength(chunkID);
  CU_ASSERT_NOT_EQUAL(chunkLen, 0);
  CU_ASSERT_TRUE(strncmp("{Hello, Server!}", chunk, chunkLen) == 0);
  free(chunk);
  GoReleaseChunk(chunkID);

  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "DONE");
  free(cmd);

  cleanRequest();
}

static int seqChar(int last) {
  int ch = last;
  for (;;) {
    ch++;
    if (ch > 255) {
      ch = 0;
    }
    if ((ch != '{') && (ch != '}')) {
      return ch;
    }
  }
}

static char* makeChunk(int len, int* lc) {
  char* c = malloc(len);
  int ch = *lc;
  for (int p = 0; p < len; p++) {
    c[p] = ch;
    ch = seqChar(ch);
  }
  *lc = ch;
  return c;
}

static void test_replace_binary(int numChunks, int chunkLen) {
  initRequest();
  createHeader("GET", "/transformbodychunks", 0, NULL);
  GoBeginRequest(id, hdrBuf);
  char* cmd = GoPollRequest(id, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "DONE");
  free(cmd);

  createResponse(numChunks * chunkLen, "text/plain");
  GoBeginResponse(rid, id, 200, hdrBuf);
  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_TRUE(strncmp("WHDR", cmd, 4) == 0);
  free(cmd);

  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "RBOD");
  free(cmd);

  int lc = 0;
  for (int c = 0; c < numChunks; c++) {
    char* nc = makeChunk(chunkLen, &lc);
    GoSendResponseBodyChunk(rid, 0, nc, chunkLen);
    free(nc);
  }
  GoSendResponseBodyChunk(rid, 1, NULL, 0);

  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_TRUE(strncmp("WBOD", cmd, 4) == 0);
  unsigned int chunkID = strtoul(cmd + 4, NULL, 16);
  free(cmd);
  char* chunk = (char*)GoGetChunk(chunkID);
  CU_ASSERT_PTR_NOT_NULL(chunk);
  unsigned int respLen = GoGetChunkLength(chunkID);
  CU_ASSERT_NOT_EQUAL(chunkLen, 0);
  CU_ASSERT_EQUAL(chunk[0], '{');
  CU_ASSERT_EQUAL(chunk[respLen - 1], '}');

  int lastCh = 0;
  for (int c = 0; c < respLen; c++) {
    int lc = chunk[c] & 0xff;
    if ((lc != '{') && (lc != '}')) {
      CU_ASSERT_EQUAL_FATAL(lc, lastCh);
      lastCh = seqChar(lastCh);
    }
  }

  free(chunk);
  GoReleaseChunk(chunkID);

  cmd = GoPollResponse(rid, 1);
  CU_ASSERT_STRING_EQUAL(cmd, "DONE");
  free(cmd);

  cleanRequest();
}

static void test_replace_response_body_binary(void) {
  test_replace_binary(1, 100);
}

static void test_replace_response_body_binary_multi(void) {
  test_replace_binary(10, 100);
}

static void test_replace_response_body_binary_larger(void) {
  test_replace_binary(10, 1000);
}

static void concurrent_test(int concurrency) {
  int* ids = (int*)malloc(sizeof(int) * concurrency);
  int* rids = (int*)malloc(sizeof(int) * concurrency);

  for (int c = 0; c < concurrency; c++) {
    ids[c] = GoCreateRequest(TEST_HANDLER);
    rids[c] = GoCreateResponse(TEST_HANDLER);
  }

  /* Start the requests. If there is some blocking somewhere, this may not work. */
  char* cmd;
  createHeader("GET", "/transformbodychunks", 0, NULL);
  for (int c = 0; c < concurrency; c++) {
    GoBeginRequest(ids[c], hdrBuf);
  }
  for (int c = 0; c < concurrency; c++) {
    cmd = GoPollRequest(ids[c], 1);
    CU_ASSERT_STRING_EQUAL(cmd, "DONE");
    free(cmd);
  }

  createResponse(10, "text/plain");
  for (int c = 0; c < concurrency; c++) {
    GoBeginResponse(rids[c], ids[c], 200, hdrBuf);
  }

  for (int c = 0; c < concurrency; c++) {
    cmd = GoPollResponse(rids[c], 1);
    CU_ASSERT_TRUE(strncmp("WHDR", cmd, 4) == 0);
    free(cmd);
    cmd = GoPollResponse(rids[c], 1);
    CU_ASSERT_STRING_EQUAL(cmd, "RBOD");
    free(cmd);
  }

  char* bod = "Hello, Server!";
  for (int c = 0; c < concurrency; c++) {
    GoSendResponseBodyChunk(rids[c], 1, bod, strlen(bod));
  }

  for (int c = 0; c < concurrency; c++) {
    cmd = GoPollResponse(rids[c], 1);
    CU_ASSERT_TRUE(strncmp("WBOD", cmd, 4) == 0);
    unsigned int chunkID = strtoul(cmd + 4, NULL, 16);
    free(cmd);
    char* chunk = (char*)GoGetChunk(chunkID);
    CU_ASSERT_PTR_NOT_NULL(chunk);
    unsigned int chunkLen = GoGetChunkLength(chunkID);
    CU_ASSERT_NOT_EQUAL(chunkLen, 0);
    CU_ASSERT_TRUE(strncmp("{Hello, Server!}", chunk, chunkLen) == 0);
    free(chunk);
    GoReleaseChunk(chunkID);

    cmd = GoPollResponse(rids[c], 1);
    CU_ASSERT_STRING_EQUAL(cmd, "DONE");
    free(cmd);
  }

  for (int c = 0; c < concurrency; c++) {
    GoFreeRequest(ids[c]);
    GoFreeResponse(rids[c]);
  }
  free(ids);
  free(rids);
}

static void test_two_concurrent_requests(void) {
  concurrent_test(2);
}

static void test_many_concurrent_requests(void) {
  concurrent_test(100);
}

int addMainTests(CU_pSuite s) {
  CU_ADD_TEST(s, test_bad_handler);
  CU_ADD_TEST(s, test_basic_request);
  CU_ADD_TEST(s, test_replace_request_body);
  CU_ADD_TEST(s, test_replace_response_body);
  CU_ADD_TEST(s, test_replace_response_body_chunks);
  CU_ADD_TEST(s, test_replace_response_body_binary);
  CU_ADD_TEST(s, test_replace_response_body_binary_multi);
  CU_ADD_TEST(s, test_replace_response_body_binary_larger);
  CU_ADD_TEST(s, test_two_concurrent_requests);
  CU_ADD_TEST(s, test_many_concurrent_requests);
  return 0;
}
