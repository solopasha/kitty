/*
 * logging.c
 * Copyright (C) 2018 Kovid Goyal <kovid at kovidgoyal.net>
 *
 * Distributed under terms of the GPL3 license.
 */

#include "data-types.h"
#include "charsets.h"
#include <stdlib.h>
#include <stdarg.h>
#include <time.h>
#include <sys/time.h>
#ifdef __APPLE__
#include <os/log.h>
#endif


static bool use_os_log = false;

void
log_error(const char *fmt, ...) {
    int n = 0;
    va_list ar;
    va_start(ar, fmt);
    n = vsnprintf(NULL, 0, fmt, ar);
    va_end(ar);
    if (n < 0) return;
    size_t size = 5 * (size_t)n + 8;
    RAII_ALLOC(char, arena, calloc(size, sizeof(char)));
    if (!arena) return;
    va_start(ar, fmt);
    n = vsnprintf(arena, size, fmt, ar);
    va_end(ar);
    char *sanbuf = arena + n + 1;

    char utf8buf[4];
    START_ALLOW_CASE_RANGE
    size_t j = 0;
    for (char *x = arena; x < arena + n; x++) {
        switch(*x) {
            case C0_EXCEPT_NL_SPACE_TAB: {
                const uint32_t ch = 0x2400 + *x;
                const unsigned sz = encode_utf8(ch, utf8buf);
                for (unsigned c = 0; c < sz; c++, j++) sanbuf[j] = utf8buf[c];
            } break;
            default:
                sanbuf[j++] = *x;
                break;
        }
    }
    sanbuf[j] = 0;
    END_ALLOW_CASE_RANGE

    if (!use_os_log) {  // Apple's os_log already records timestamps
        struct timeval tv;
        gettimeofday(&tv, NULL);
        struct tm stack_tmp;
        struct tm *tmp = localtime_r(&tv.tv_sec, &stack_tmp);
        if (tmp) {
            char tbuf[256] = {0}, buf[256] = {0};
            if (strftime(buf, sizeof(buf), "%j %H:%M:%S.%%06u", tmp) != 0) {
                snprintf(tbuf, sizeof(tbuf), buf, tv.tv_usec);
                fprintf(stderr, "[%s] ", tbuf);
            }
        }
    }
#ifdef __APPLE__
    if (use_os_log) os_log(OS_LOG_DEFAULT, "%{public}s", sanbuf);
#endif
    if (!use_os_log) fprintf(stderr, "%s\n", sanbuf);
#undef bufprint
}

static PyObject*
log_error_string(PyObject *self UNUSED, PyObject *args) {
    const char *msg;
    if (!PyArg_ParseTuple(args, "s", &msg)) return NULL;
    log_error("%s", msg);
    Py_RETURN_NONE;
}

static PyObject*
set_use_os_log(PyObject *self UNUSED, PyObject *args) {
    use_os_log = PyObject_IsTrue(args) ? true : false;
    Py_RETURN_NONE;
}

static PyMethodDef module_methods[] = {
    METHODB(log_error_string, METH_VARARGS),
    METHODB(set_use_os_log, METH_O),
    {NULL, NULL, 0, NULL}        /* Sentinel */
};

bool
init_logging(PyObject *module) {
    if (PyModule_AddFunctions(module, module_methods) != 0) return false;
#ifdef __APPLE__
    if (getenv("KITTY_LAUNCHED_BY_LAUNCH_SERVICES") != NULL) use_os_log = true;
#endif
    return true;
}
