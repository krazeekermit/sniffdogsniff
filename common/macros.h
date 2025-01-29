#ifndef MACROS_H
#define MACROS_H

#define packed_struct struct __attribute__((__packed__))

#define STREAM_HEX(O, A, L) \
{ \
    for (int i = 0; i < L; i++) { \
        uint8_t hi = (A[i] >> 4) & 0x0f; \
        uint8_t lo = (A[i] & 0x0f); \
        O << (char) (hi > 9 ? ('a' + (hi - 10)) : ('0' + hi)); \
        O << (char) (lo > 9 ? ('a' + (lo - 10)) : ('0' + lo)); \
    } \
}

#define STREAM_HEX_REVERSE(O, A, L) \
{ \
    for (int i = L-1; i; i--) { \
        uint8_t hi = (A[i] >> 4) & 0x0f; \
        uint8_t lo = (A[i] & 0x0f); \
        O << (char) (hi > 9 ? ('a' + (hi - 10)) : ('0' + hi)); \
        O << (char) (lo > 9 ? ('a' + (lo - 10)) : ('0' + lo)); \
    } \
}

#endif // MACROS_H
