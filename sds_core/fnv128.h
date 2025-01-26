/*************************** FNV128.h *************************/
//***************** See RFC NNNN for details *******************//
/* Copyright (c) 2016, 2023, 2024 IETF Trust and the persons
 * identified as authors of the code.  All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 *
 * *  Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 *
 * *  Redistributions in binary form must reproduce the above
 *    copyright notice, this list of conditions and the following
 *    disclaimer in the documentation and/or other materials provided
 *    with the distribution.
 *
 * *  Neither the name of Internet Society, IETF or IETF Trust, nor
 *    the names of specific contributors, may be used to endorse or
 *    promote products derived from this software without specific
 *    prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND
 * CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES,
 * INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS
 * BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
 * EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED
 * TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
 * ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR
 * TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF
 * THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 */

#ifndef _FNV128_H_
#define _FNV128_H_

/*
 *  Description:
 *      This file provides headers for the 128-bit version of the
 *      FNV-1a non-cryptographic hash algorithm.
 */

#include <stdint.h>
#define FNV128size (128/8)

#if defined(__x86_64__) || defined(__amd64__) || defined(__aarch64__)
#define FNV_64bitIntegers
#endif

/* If you do not have the ISO standard stdint.h header file, then
 * you must typedef the following types:
 *
 *    type              meaning
 *  uint64_t    unsigned 64 bit integer (ifdef FNV_64bitIntegers)
 *  uint32_t    unsigned 32 bit integer
 *  uint16_t    unsigned 16 bit integer
 *  uint8_t     unsigned 8 bit integer (i.e., unsigned char)
 */

enum {  /* State value bases for context->Computed */
    FNVinited = 22,
    FNVcomputed = 76,
    FNVemptied = 220,
    FNVclobber = 122 /* known bad value for testing */
};

/* Deltas to assure distinct state values for different lengths */
enum {
   FNV32state = 1,
   FNV64state = 3,
   FNV128state = 5,
   FNV256state = 7,
   FNV512state = 11,
   FNV1024state = 13
};

//******************************************************************
//  All FNV functions provided return as integer as follows:
//       0 -> success
//      >0 -> error as listed below
//
enum {    /* success and errors */
    fnvSuccess = 0,
    fnvNull,          /* Null pointer parameter */
    fnvStateError,    /* called Input after Result or before Init */
    fnvBadParam       /* passed a bad parameter */
};

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
