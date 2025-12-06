#ifndef SEARCHENTRY_H
#define SEARCHENTRY_H

#include "simhash.h"

#include <map>
#include <cstdint>
#include <iostream>
#include <ctime>
#include <vector>
#include <array>
#include <openssl/sha.h>

#define MAX_SEARCH_ENTRY_SZ 1104
#define METRICS_LEN       4

class SearchEntry {

public:
    enum Type {
        SITE =  0,
        IMAGE = 1,
        VIDEO = 2
    };

    struct Hash
    {
        Hash();
        Hash(uint8_t *_data);

        bool operator<(const Hash &hash2) const;

        /* member (actual hash data) */
        uint8_t hash[SHA256_DIGEST_LENGTH];
    };

    SearchEntry();
    SearchEntry(const std::string title, const std::string url, Type type = Type::SITE, std::map<uint8_t, std::string> properties = {});
    ~SearchEntry();

    void addProperty(uint8_t idx, const std::string value);
    const std::string getProperty(uint8_t idx);
    void removeProperty(uint8_t idx);

    void reHash();
    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);

    bool matchesQuery(std::vector<std::string> tokens);

    Hash getHash() const;
    std::string getTitle() const;
    std::string getUrl() const;

    SimHash getSimHash() const;

    friend std::ostream &operator<< (std::ostream &os, SearchEntry const &se);

    Type getType() const;

private:
    Hash hash;
    SimHash simHash;
    std::string title;
    std::string url;
    Type type;
    std::map<uint8_t, std::string> properties;
};

#endif // SEARCHENTRY_H
