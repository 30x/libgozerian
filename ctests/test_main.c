#include <stdio.h>
#include <string.h>
#include <CUnit/Basic.h>
#include <libgozerian.h>
#include "ctests.h"

static int init_suite(void) {
  char* err = GoCreateHandler(TEST_HANDLER, "urn:weaver-proxy:unit-test");
  if (err != NULL) {
    fprintf(stderr, "Error creating test handler: %s\n", err);
    return -1;
  }
  return 0;
}

static int clean_suite(void) {
  GoDestroyHandler(TEST_HANDLER);
  return 0;
}

int main() {
  CU_pSuite pSuite = NULL;

  /* initialize the CUnit test registry */
  if (CUE_SUCCESS != CU_initialize_registry())
    return CU_get_error();

  /* add a suite to the registry */
  pSuite = CU_add_suite("libgozerian", init_suite, clean_suite);
  if (NULL == pSuite) {
    CU_cleanup_registry();
    return CU_get_error();
  }

  if (addMainTests(pSuite) != 0) {
    CU_cleanup_registry();
    return CU_get_error();
  }

  /* Run all tests using the CUnit Basic interface */
  CU_basic_set_mode(CU_BRM_VERBOSE);
  CU_basic_run_tests();
  CU_cleanup_registry();
  return CU_get_error();
}
