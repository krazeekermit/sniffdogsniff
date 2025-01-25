/*************************** FNV128.h *************************/
/**************** See RFC NNNN for details. *******************/
/*
 * Copyright (c) 2016, 2023 IETF Trust and the persons identified
 * as authors of the code.  All rights reserved.
 * See fnv-private.h for terms of use and redistribution.
 */

#ifndef _FNV128_H_
#define _FNV128_H_

/*
 *  Description:
 *      This file provides headers for the 128-bit version of the
 *      FNV-1a non-cryptographic hash algorithm.
 */

#include "FNVconfig.h"

#include <stdint.h>
#define FNV128size (128/8)

/* If you do not have the ISO standard stdint.h header file, then
 * you must typedef the following types:
 *
 *    type              meaning
 *  uint64_t    unsigned 64 bit integer (ifdef FNV_64bitIntegers)
 *  uint32_t    unsigned 32 bit integer
 *  uint16_t    unsigned 16 bit integer
 *  uint8_t     unsigned 8 bit integer (i.e., unsigned char)
 */

#include "FNVErrorCodes.h"

/*
 *  This structure holds context information for an FNV128 hash
 */
#ifdef FNV_64bitIntegers
    /* version if 64 bit integers supported */
typedef struct FNV128context_s {
        int Computed;  /* state */
        uint32_t Hash[FNV128size/4];
} FNV128context;

#else
    /* version if 64 bit integers NOT supported */

typedef struct FNV128context_s {
        int Computed;  /* state */
        uint16_t Hash[FNV128size/2];
} FNV128context;

#endif /* FNV_64bitIntegers */

/*
 *  Function Prototypes
 *    FNV128string: hash a zero terminated string not including
 *                  the terminating zero
 *    FNV128block: FNV128 hash a specified length byte vector
 *    FNV128init: initializes an FNV128 context
 *    FNV128initBasis: initializes an FNV128 context with a
 *                     provided basis
 *    FNV128blockin: hash in a specified length byte vector
 *    FNV128stringin: hash in a zero terminated string not
 *                    including the zero
 *    FNV128result: returns the hash value
 *
 *    Hash is returned as an array of 8-bit unsigned integers
 */

#ifdef __cplusplus
extern "C" {
#endif

/* FNV128 */
extern int FNV128string ( const char *in,
                          uint8_t out[FNV128size] );
extern int FNV128block ( const void *in,
                         long int length,
                         uint8_t out[FNV128size] );
extern int FNV128init ( FNV128context * const );
extern int FNV128initBasis ( FNV128context * const,
                             const uint8_t basis[FNV128size] );
extern int FNV128blockin ( FNV128context * const,
                           const void *in,
                           long int length );
extern int FNV128stringin ( FNV128context * const,
                            const char *in );
extern int FNV128result ( FNV128context * const,
                          uint8_t out[FNV128size] );

#ifdef __cplusplus
}
#endif

#endif /* _FNV128_H_ */
