#ifndef SEARCHENTRIESDB_H
#define SEARCHENTRIESDB_H

#include "simhash.h"

#include <map>
#include <cstdint>
#include <iostream>
#include <ctime>
#include <vector>
#include <array>
#include <openssl/sha.h>
#include <db5.3/db.h>

#define MAX_SEARCH_ENTRY_SZ 1104
#define METRICS_LEN       4

typedef std::array<uint8_t, SHA256_DIGEST_LENGTH> SearchEntryHash256;

enum SearchEntryType {
    SITE =  0,
    IMAGE = 1,
    VIDEO = 2
};

class SearchEntry {

public:
    SearchEntry();
    SearchEntry(const std::string title, const std::string url, SearchEntryType type = SearchEntryType::SITE, std::map<uint8_t, std::string> properties = {});
    ~SearchEntry();

    void addProperty(uint8_t idx, const std::string value);
    const std::string getProperty(uint8_t idx);
    void removeProperty(uint8_t idx);

    void reHash();
    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);

    friend std::ostream &operator<< (std::ostream &os, SearchEntry const &se);

    SearchEntryHash256 getHash() const;
    std::string getTitle() const;
    std::string getUrl() const;

    SimHash getSimHash() const;

private:
    SearchEntryHash256 hash;
    SimHash simHash;
    std::string title;
    std::string url;
    SearchEntryType type;
    std::map<uint8_t, std::string> properties;

    void evaluateDistances();
};

/*****************************************************************

        SearchEntriesDB

*****************************************************************/

class SearchEntriesDB
{
public:
    SearchEntriesDB();
    ~SearchEntriesDB();

    void open(const char *db_path);

    void insertResult(SearchEntry &se);
    int getEntriesForBroadcast(std::vector<SearchEntry> &list);
    void doSearch(std::vector<SearchEntry> &entries, std::string query);
    void flush();
    void close();

private:
    std::map<SearchEntryHash256, time_t> timestamps;
    DB *dbp;

    SearchEntry getByHash(const SearchEntryHash256 &hash);
    void modified();
};

#endif // SEARCHENTRIESDB_H
