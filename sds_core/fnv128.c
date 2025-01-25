/***************************** FNV128.c ***************************/
/******************** See RFC NNNN for details ********************/
/* Copyright (c) 2016, 2023 IETF Trust and the persons identified as
 * authors of the code.  All rights
 * See fnv-private.h for terms of use and redistribution.
 */

/* This file implements the FNV (Fowler, Noll, Vo) non-cryptographic
 * hash function FNV-1a for 128-bit hashes.
 */

#ifndef _FNV128_C_
#define _FNV128_C_

#include "fnv-private.h"
#include "FNV128.h"

/* common code for 64 and 32 bit modes */

/* FNV128 hash a null terminated string  (64/32 bit)
 ******************************************************************/
int FNV128string ( const char *in, uint8_t out[FNV128size] )
{
    FNV128context    ctx;
    int              err;

    if ( (err = FNV128init ( &ctx )) != fnvSuccess )
        return err;
    if ( (err = FNV128stringin ( &ctx, in )) != fnvSuccess )
        return err;
    return FNV128result (&ctx, out);
}   /* end FNV128string */

/* FNV128 hash a counted block  (64/32 bit)
 ******************************************************************/
int FNV128block ( const void *in,
                  long int length,
                  uint8_t out[FNV128size] )
{
    FNV128context    ctx;
    int              err;

   if ( (err = FNV128init ( &ctx )) != fnvSuccess )
       return err;
   if ( (err = FNV128blockin ( &ctx, in, length )) != fnvSuccess )
       return err;
    return FNV128result ( &ctx, out );
}   /* end FNV128block */


/******************************************************************
 *        START VERSION FOR WHEN YOU HAVE 64 BIT ARITHMETIC       *
 ******************************************************************/
#ifdef FNV_64bitIntegers

/* 128 bit FNV_prime = 2^88 + 2^8 + 0x3b */
/* 0x00000000 01000000 00000000 0000013B */
#define FNV128primeX 0x013B
#define FNV128shift 24

/* 0x6C62272E 07BB0142 62B82175 6295C58D */
#define FNV128basis0 0x6C62272E
#define FNV128basis1 0x07BB0142
#define FNV128basis2 0x62B82175
#define FNV128basis3 0x6295C58D

/******************************************************************
 *         Set of init, input, and output functions below         *
 *         to incrementally compute FNV128                        *
 ******************************************************************/

/* initialize context  (64 bit)
 ******************************************************************/
int FNV128init ( FNV128context * const ctx )
{
    if ( !ctx )
        return fnvNull;

    ctx->Hash[0] = FNV128basis0;
    ctx->Hash[1] = FNV128basis1;
    ctx->Hash[2] = FNV128basis2;
    ctx->Hash[3] = FNV128basis3;
    ctx->Computed = FNVinited+FNV128state;
    return fnvSuccess;
}   /* end FNV128init */

/* initialize context with a provided basis  (64 bit)
 ******************************************************************/
int FNV128initBasis ( FNV128context * const ctx,
                      const uint8_t basis[FNV128size] )
{
    int       i;
    const uint8_t   *ui8p;
    uint32_t  temp;

    if ( !ctx )
        return fnvNull;

#ifdef FNV_BigEndian
    ui8p = basis;
    for ( i=0; i < FNV128size/4; ++i )
        {
        temp = (*ui8p++)<<8;
        temp = (temp + *ui8p++)<<8;
            temp = (temp + *ui8p++)<<8;
        ctx->Hash[i] = temp +  *ui8p;
        }
#else
    ui8p = basis + ( FNV128size/4 - 1 );
    for ( i=0; i < FNV128size/4; ++i )
        {
            temp = (*ui8p--)<<8;
            temp = (temp + *ui8p--)<<8;
            temp = (temp + *ui8p--)<<8;
        ctx->Hash[i] = temp +  *ui8p;
        }
#endif
    ctx->Computed = FNVinited+FNV128state;
    return fnvSuccess;
}   /* end FNV128initBasis */

/* hash in a counted block  (64 bit)
 ******************************************************************/
int FNV128blockin ( FNV128context * const ctx,
                    const void *vin,
                    long int length )
{
    const uint8_t *in = (const uint8_t*)vin;
    uint64_t    temp[FNV128size/4];
    uint64_t    temp2[2];
    int         i;

    if ( !ctx || !in )
        return fnvNull;

    if ( length < 0 )
        return fnvBadParam;
    switch ( ctx->Computed )
        {
        case FNVinited+FNV128state:
            ctx->Computed = FNVcomputed+FNV128state;
        case FNVcomputed+FNV128state:
            break;
        default:
            return fnvStateError;
        }
    for ( i=0; i<FNV128size/4; ++i )
         temp[i] = ctx->Hash[i];
    for ( ; length > 0; length-- )
        {
        /* temp = FNV128prime * ( temp ^ *in++ ); */
        temp2[1] = temp[3] << FNV128shift;
        temp2[0] = temp[2] << FNV128shift;
        temp[3] = FNV128primeX * ( temp[3] ^ *in++ );
        temp[2] *= FNV128primeX;
        temp[1] = temp[1] * FNV128primeX + temp2[1];
        temp[0] = temp[0] * FNV128primeX + temp2[0];
        temp[2] += temp[3] >> 32;
        temp[3] &= 0xFFFFFFFF;
        temp[1] += temp[2] >> 32;
        temp[2] &= 0xFFFFFFFF;
        temp[0] += temp[1] >> 32;
        temp[1] &= 0xFFFFFFFF;
        }
    for ( i=0; i<FNV128size/4; ++i )
        ctx->Hash[i] = temp[i];
    return fnvSuccess;
}   /* end FNV128input */

/* hash in a string  (64 bit)
 ******************************************************************/
int FNV128stringin ( FNV128context * const ctx, const char *in )
{
    uint64_t   temp[FNV128size/4];
    uint64_t   temp2[2];
    int        i;
    uint8_t    ch;

    if ( !ctx || !in )
        return fnvNull;

    switch ( ctx->Computed )
        {
        case FNVinited+FNV128state:
            ctx->Computed = FNVcomputed+FNV128state;
        case FNVcomputed+FNV128state:
            break;
        default:
             return fnvStateError;
         }
    for ( i=0; i<FNV128size/4; ++i )
         temp[i] = ctx->Hash[i];
    while ( (ch = (uint8_t)*in++) )
        {
        /* temp = FNV128prime * ( temp ^ ch ); */
        temp2[1] = temp[3] << FNV128shift;
        temp2[0] = temp[2] << FNV128shift;
        temp[3] = FNV128primeX * ( temp[3] ^ *in++ );
        temp[2] *= FNV128primeX;
        temp[1] = temp[1] * FNV128primeX + temp2[1];
        temp[0] = temp[0] * FNV128primeX + temp2[0];
        temp[2] += temp[3] >> 32;
        temp[3] &= 0xFFFFFFFF;
        temp[1] += temp[2] >> 32;
        temp[2] &= 0xFFFFFFFF;
        temp[0] += temp[1] >> 32;
        temp[1] &= 0xFFFFFFFF;
        }
    for ( i=0; i<FNV128size/4; ++i )
        ctx->Hash[i] = temp[i];
    return fnvSuccess;
}   /* end FNV128stringin */

/* return hash  (64 bit)
 ******************************************************************/
int FNV128result ( FNV128context * const ctx,
                   uint8_t out[FNV128size] )
{
    int     i;

    if ( !ctx || !out )
        return fnvNull;

    if ( ctx->Computed != FNVcomputed+FNV128state )
        return fnvStateError;
    for ( i=0; i<FNV128size/4; ++i )
        {
#ifdef FNV_BigEndian
        out[15-4*i] = ctx->Hash[i];
        out[14-4*i] = ctx->Hash[i] >> 8;
        out[13-4*i] = ctx->Hash[i] >> 16;
        out[12-4*i] = ctx->Hash[i] >> 24;
#else
        out[4*i] = ctx->Hash[i];
        out[4*i+1] = ctx->Hash[i] >> 8;
        out[4*i+2] = ctx->Hash[i] >> 16;
        out[4*i+3] = ctx->Hash[i] >> 24;
#endif
        ctx -> Hash[i] = 0;
        }
    ctx->Computed = FNVemptied+FNV128state;
    return fnvSuccess;
}   /* end FNV128result */

/*****************************************************************
 *        END VERSION FOR WHEN YOU HAVE 64 BIT ARITHMETIC        *
 *****************************************************************/
#else    /*  FNV_64bitIntegers */
/*****************************************************************
 *     START VERSION FOR WHEN YOU ONLY HAVE 32-BIT ARITHMETIC    *
 *****************************************************************/

/* 128 bit FNV_prime = 2^88 + 2^8 + 0x3b */
/* 0x00000000 01000000 00000000 0000013B */
#define FNV128primeX 0x013B
#define FNV128shift 8

/* 0x6C62272E 07BB0142 62B82175 6295C58D */
uint16_t FNV128basis[FNV128size/2] =
         { 0x6C62, 0x272E, 0x07BB, 0x0142,
           0x62B8, 0x2175, 0x6295, 0xC58D };

/******************************************************************
 *         Set of init, input, and output functions below         *
 *         to incrementally compute FNV128                        *
 ******************************************************************/

/* initialize context  (32 bit)
 ******************************************************************/
int FNV128init ( FNV128context *ctx )
{
    int     i;

    if ( !ctx )
        return fnvNull;

    for ( i=0; i<FNV128size/2; ++i )
        ctx->Hash[i] = FNV128basis[i];
    ctx->Computed = FNVinited+FNV128state;
    return fnvSuccess;
}   /* end FNV128init */

/* initialize context with a provided basis  (32 bit)
 ******************************************************************/
int FNV128initBasis ( FNV128context *ctx,
                      const uint8_t basis[FNV128size] )
{
    int      i;
    const uint8_t  *ui8p;
    uint32_t temp;

    if ( !ctx )
        return fnvNull;

#ifdef FNV_BigEndian
    ui8p = basis;
    for ( i=0; i < FNV128size/2; ++i )
        {
        temp = *ui8p++;
        ctx->Hash[i] = ( temp<<8 ) + (*ui8p++);
        }
#else
    ui8p = basis + (FNV128size/2 - 1);
    for ( i=0; i < FNV128size/2; ++i )
        {
        temp = *ui8p--;
        ctx->Hash[i] = ( temp<<8 ) +  (*ui8p--);
        }
#endif
    ctx->Computed = FNVinited+FNV128state;
    return fnvSuccess;
}   /* end FNV128initBasis */

/* hash in a counted block  (32 bit)
 *****************************************************************/
int FNV128blockin ( FNV128context *ctx,
                    const void *vin,
                    long int length )
{
    const uint8_t *in = (const uint8_t*)vin;
    uint32_t   temp[FNV128size/2];
    uint32_t   temp2[3];
    int        i;

    if ( !ctx || !in )
        return fnvNull;

    if ( length < 0 )
        return fnvBadParam;
    switch ( ctx->Computed )
        {
        case FNVinited+FNV128state:
            ctx->Computed = FNVcomputed+FNV128state;
        case FNVcomputed+FNV128state:
            break;
        default:
            return fnvStateError;
        }
    for ( i=0; i<FNV128size/2; ++i )
         temp[i] = ctx->Hash[i];
    for ( ; length > 0; length-- )
        {
        /* temp = FNV128prime * ( temp ^ *in++ ); */
        temp[7] ^= *in++;
        temp2[2] = temp[7] << FNV128shift;
        temp2[1] = temp[6] << FNV128shift;
        temp2[0] = temp[5] << FNV128shift;
        for ( i=0; i<8; ++i )
            temp[i] *= FNV128primeX;
        temp[2] += temp2[2];
        temp[1] += temp2[1];
        temp[0] += temp2[0];
        for ( i=7; i>0; --i )
            {
            temp[i-1] += temp[i] >> 16;
            temp[i] &= 0xFFFF;
            }
        }
    for ( i=0; i<FNV128size/2; ++i )
        ctx->Hash[i] = temp[i];
    return fnvSuccess;
}   /* end FNV128blockin */

/* hash in a string  (32 bit)
 ******************************************************************/
int FNV128stringin ( FNV128context *ctx,
                     const char *in )
{
    uint32_t   temp[FNV128size/2];
    uint32_t   temp2[3];
    int        i;
    uint8_t    ch;

    if ( !ctx || !in )
        return fnvNull;

    switch ( ctx->Computed )
        {
        case FNVinited+FNV128state:
            ctx->Computed = FNVcomputed+FNV128state;
        case FNVcomputed+FNV128state:
            break;
        default:
             return fnvStateError;
         }
    for ( i=0; i<FNV128size/2; ++i )
         temp[i] = ctx->Hash[i];
    while ( (ch = (uint8_t)*in++) )
        {
        /* temp = FNV128prime * ( temp ^ *in++ ); */
        temp[7] ^= ch;
        temp2[2] = temp[7] << FNV128shift;
        temp2[1] = temp[6] << FNV128shift;
        temp2[0] = temp[5] << FNV128shift;
        for ( i=0; i<8; ++i )
            temp[i] *= FNV128primeX;
        temp[2] += temp2[2];
        temp[1] += temp2[1];
        temp[0] += temp2[0];
        for ( i=7; i>0; --i )
            {
            temp[i-1] += temp[i] >> 16;
            temp[i] &= 0xFFFF;
            }
         }
    for ( i=0; i<FNV128size/2; ++i )
        ctx->Hash[i] = temp[i];
    return fnvSuccess;
}   /* end FNV128stringin */

/* return hash  (32 bit)
 ******************************************************************/
int FNV128result ( FNV128context *ctx,
                   uint8_t out[FNV128size] )
{
    int    i;

    if ( !ctx || !out )
        return fnvNull;

    if ( ctx->Computed != FNVcomputed+FNV128state )
        return fnvStateError;
    for ( i=0; i<FNV128size/2; ++i )
        {
#ifdef FNV_BigEndian
        out[15-2*i] = ctx->Hash[i];
        out[14-2*i] = ctx->Hash[i] >> 8;
#else
        out[2*i] = ctx->Hash[i];
        out[2*i+1] = ctx->Hash[i] >> 8;
#endif
        ctx -> Hash[i] = 0;
        }
    ctx->Computed = FNVemptied+FNV128state;
    return fnvSuccess;
}   /* end FNV128result */

#endif    /*  FNV_64bitIntegers */
/*******************************************************************
 *        END VERSION FOR WHEN YOU ONLY HAVE 32-BIT ARITHMETIC     *
 *******************************************************************/

#endif    /* _FNV128_C_ */
