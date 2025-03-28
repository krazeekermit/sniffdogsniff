#include "simhash.h"

#include "common/stringutil.h"

#include "fnv128.h"

#include <map>
#include <cstring>
#include <algorithm>
#include <cctype>
#include <string>

#include <climits>

struct wordhash {
    int weight;
    unsigned char *hash;
};

SimHash::SimHash(std::string str)
{
    std::vector<std::string> tokens = tokenize(str, " \n\r", ".:,;()[]{}#@");
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
    for (int i = 0; i < KAD_ID_LENGTH; i++) {
        for (unsigned char n = ax.id[i]; n; hamming++) {
            n &= n - 1;
        }
    }

    return hamming;
}

void SimHash::read(SdsBytesBuf &buf)
{
    buf.readBytes(this->id.id, KAD_ID_LENGTH);
}

void SimHash::write(SdsBytesBuf &buf)
{
    buf.writeBytes(this->id.id, KAD_ID_LENGTH);
}

void SimHash::init(std::vector<std::string> &tokens)
{
    int i, j;
    std::map<std::string, wordhash> tokenMults;
    for (i = 0; i < tokens.size(); i++) {
        std::string tok = tokens[i];
        if (tokenMults.find(tok) == tokenMults.end()) {
            unsigned char *hash = new unsigned char[KAD_ID_LENGTH];

            FNV128string(tok.c_str(), hash);

            tokenMults[tok].weight = 0;
            tokenMults[tok].hash = hash;
        }
        tokenMults[tok].weight += 1;
    }

    char simWeights[KAD_ID_BIT_LENGTH];
    memset(simWeights, 0, sizeof(simWeights));

    for (auto it = tokenMults.begin(); it != tokenMults.end(); it++) {
        unsigned char *hash = it->second.hash;
        for (i = 0; i < KAD_ID_LENGTH; i++) {
            char ih = hash[i];
            for (j = 0; j < 8; j++) {
                int no = ((ih & (0x80 >> j)) == 0 ? -1 : 1) * it->second.weight;
                simWeights[i*8 + j] += no;
            }
        }

        delete[] hash;
    }

    memset(this->id.id, 0, sizeof(this->id.id));
    for (i = 0; i < KAD_ID_LENGTH; i++) {
        for (j = 0; j < 8; j++) {
            if (simWeights[i*8 + j] > 0) {
                this->id.id[i] |= (0x80 >> j);
            }
        }
    }
}
