#ifndef C_TESTS_H
#define C_TESTS_H

#include <CUnit/Basic.h>

#ifdef DMALLOC
#include <dmalloc.h>
#endif

#define TEST_HANDLER "test"

extern int addMainTests(CU_pSuite suite);

#endif
