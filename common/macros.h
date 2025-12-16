#ifndef MACROS_H
#define MACROS_H

#define packed_struct struct __attribute__((__packed__))

#ifdef __OpenBSD__
#if BYTE_ORDER == BIG_ENDIAN
#define SDS_BIG_ENDIAN
#endif
#else
#if (__BYTE_ORDER__ == __ORDER_BIG_ENDIAN__)
#define SDS_BIG_ENDIAN
#endif
#endif

#endif // MACROS_H
