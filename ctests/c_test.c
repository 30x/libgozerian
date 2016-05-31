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
  free(err);
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

int addMainTests(CU_pSuite s) {
  CU_ADD_TEST(s, test_bad_handler);
  CU_ADD_TEST(s, test_basic_request);
  return 0;
}
