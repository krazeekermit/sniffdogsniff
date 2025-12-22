#include "simhash.h"

#include "common/stringutil.h"
#include "common/macros.h"

#include <map>
#include <cstring>
#include <algorithm>
#include <cctype>
#include <string>

#include <climits>

/* 128 bit FNV_prime = 2^88 + 2^8 + 0x3b */
/* 0x00000000 01000000 00000000 0000013B */
#define FNV128primeX 0x013B
#define FNV128shift 24

/* 0x6C62272E 07BB0142 62B82175 6295C58D */
#define FNV128basis0 0x6C62272E
#define FNV128basis1 0x07BB0142
#define FNV128basis2 0x62B82175
#define FNV128basis3 0x6295C58D

static int fnv128String(const char *in, uint8_t *out)
{
    if (!in || !out) {
        return -1;
    }

    uint64_t   temp[FNV128_SIZE/4];
    uint64_t   temp2[2];
    int i;

    // init basis
    temp[0] = FNV128basis0;
    temp[1] = FNV128basis1;
    temp[2] = FNV128basis2;
    temp[3] = FNV128basis3;

    while ((uint8_t)*in++) {
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

    for ( i=0; i<FNV128_SIZE/4; ++i ) {
#ifdef SDS_BIG_ENDIAN
        out[15-4*i] = temp[i];
        out[14-4*i] = temp[i] >> 8;
        out[13-4*i] = temp[i] >> 16;
        out[12-4*i] = temp[i] >> 24;
#else
        out[4*i] = temp[i];
        out[4*i+1] = temp[i] >> 8;
        out[4*i+2] = temp[i] >> 16;
        out[4*i+3] = temp[i] >> 24;
#endif
    }

    return 0;
}

struct wordhash {
    int weight;
    uint8_t hash[FNV128_SIZE];
};

SimHash::SimHash(std::string str)
{
    std::vector<std::string> tokens = StringUtil::tokenize(str, " \n\r", ".:,;()[]{}#@");
    this->init(tokens);
}

SimHash::SimHash(std::vector<std::string> &tokens)
{
    this->init(tokens);
}

KadId SimHash::getId() const
{
    return id;
}

int SimHash::distance(SimHash &other)
{
    KadId ax = this->id - other.id;
    int hamming = 0;
    for (int i = 0; i < FNV128_SIZE; i++) {
        for (unsigned char n = ax.id[i]; n; hamming++) {
            n &= n - 1;
        }
    }

    return hamming;
}

void SimHash::read(SdsBytesBuf &buf)
{
    buf.readBytes(this->id.id, FNV128_SIZE);
}

void SimHash::write(SdsBytesBuf &buf)
{
    buf.writeBytes(this->id.id, FNV128_SIZE);
}

void SimHash::init(std::vector<std::string> &tokens)
{
    int i, j;
    std::map<std::string, wordhash> tokenMults;
    for (i = 0; i < tokens.size(); i++) {
        if (tokenMults.find(tokens[i]) == tokenMults.end()) {
            uint8_t *hash = tokenMults[tokens[i]].hash;
            tokenMults[tokens[i]].weight = 0;

            fnv128String(tokens[i].c_str(), hash);
        }
        tokenMults[tokens[i]].weight += 1;
    }

    char simWeights[FNV128_BIT_SIZE];
    memset(simWeights, 0, sizeof(simWeights));

    for (auto it = tokenMults.begin(); it != tokenMults.end(); it++) {
        uint8_t *hash = it->second.hash;
        for (i = 0; i < FNV128_SIZE; i++) {
            char ih = hash[i];
            for (j = 0; j < 8; j++) {
                int no = ((ih & (0x80 >> j)) == 0 ? -1 : 1) * it->second.weight;
                simWeights[i*8 + j] += no;
            }
        }
    }

    memset(this->id.id, 0, sizeof(this->id.id));
    for (i = 0; i < FNV128_SIZE; i++) {
        for (j = 0; j < 8; j++) {
            if (simWeights[i*8 + j] > 0) {
                this->id.id[i] |= (0x80 >> j);
            }
        }
    }
}
