#include "simhash.h"

#include <map>
#include <cstring>
#include <algorithm>
#include <cctype>
#include <string>

struct wordhash {
    int weight;
    unsigned char *hash;
};

static inline bool toSkipChar(char c)
{
    return c == '.' || c == ',' || c == ';' || c == ':' || c == '(' || c == ')' ||
           c == '[' || c == ']' || c == '{' || c == '}' || c == '#' || c == '@';
}

static void strclean(std::string &str)
{
    size_t i;
    for (i = 0; i < str.size(); i++) {
        str[i] = std::tolower(str[i]);
    }

    for (i = 0; i < str.size(); i++) {
        if (!toSkipChar(str[i])) {
            break;
        }
    }
    str.erase(0, i);

    size_t lastIdx = str.size() - 1;
    for (i = lastIdx; i >= 0; i--) {
        if (!toSkipChar(str[i])) {
            break;
        }
    }
    str.erase(i+1, lastIdx);
}

void SimHash::update(std::string str)
{
    size_t pos = 0;
    while ((pos = str.find(" ")) != std::string::npos) {
        std::string tok = str.substr(0, pos);
        strclean(tok);
        if (tok.size()) {
            this->tokens.push_back(tok);
        }
        str.erase(0, pos + 1);
    }

    strclean(str);
    if (str.size()) {
        this->tokens.push_back(str);
    }
}

KadId SimHash::digest() const
{
    int i, j;
    std::map<std::string, wordhash> tokenMults;
    for (i = 0; i < this->tokens.size(); i++) {
        std::string tok = this->tokens[i];
        if (tokenMults.find(tok) == tokenMults.end()) {
            unsigned char *hash = new unsigned char[SHA_DIGEST_LENGTH];
            SHA_CTX ctx;
            SHA1_Init(&ctx);
            SHA1_Update(&ctx, tok.c_str(), tok.size());
            SHA1_Final(hash, &ctx);

            tokenMults[tok].weight = 0;
            tokenMults[tok].hash = hash;
        }
        tokenMults[tok].weight += 1;
    }

    char simWeights[SHA_DIGEST_LENGTH*8];
    memset(simWeights, 0, sizeof(simWeights));

    for (auto it = tokenMults.begin(); it != tokenMults.end(); it++) {
        unsigned char *hash = it->second.hash;
        for (i = 0; i < SHA_DIGEST_LENGTH; i++) {
            char ih = hash[i];
            for (j = 0; j < 8; j++) {
                int no = ((ih & (0x80 >> j)) == 0 ? -1 : 1) * it->second.weight;
                simWeights[i*8 + j] += no;
            }
        }

        delete[] hash;
    }

    KadId id;
    for (i = 0; i < SHA_DIGEST_LENGTH; i++) {
        for (j = 0; j < 8; j++) {
            if (simWeights[i*8 + j] > 0) {
                id.id[i] |= (0x80 >> j);
            }
        }
    }

    return id;
}

const KadId SimHash::digest(std::string str)
{
    SimHash sim;
    sim.update(str);
    return sim.digest();
}
