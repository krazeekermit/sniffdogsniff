#ifndef SEARCHENTRIESDB_H
#define SEARCHENTRIESDB_H

#include "kademlia/kadbucket.h"
#include "msgpack11/msgpack11.h"

#include <map>
#include <cstdint>
#include <iostream>
#include <ctime>
#include <vector>
#include <array>
#include <openssl/sha.h>
#include <db.h>

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
    SearchEntry(const std::string title, const std::string url, SearchEntryType type, std::map<uint8_t, std::string> properties = {});
    ~SearchEntry();

    void addProperty(uint8_t idx, const std::string value);
    const std::string getProperty(uint8_t idx);
    void removeProperty(uint8_t idx);

    void reHash();
    uint8_t *toBytes() const;
    int fromBytes(uint8_t *buf);

    friend std::ostream &operator<< (std::ostream &os, SearchEntry const &se);

    SearchEntryHash256 getHash() const;
    std::string getTitle() const;
    std::string getUrl() const;

    bool unpack(msgpack11::MsgPack &obj);
    void pack(msgpack11::MsgPack &obj);

    KadId *getMetrics() const;

    static int evaluateMetrics(KadId metrics[METRICS_LEN], const char *query);

private:
    SearchEntryHash256 hash;
    KadId metrics[METRICS_LEN];
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

    void insertResult(const SearchEntry &se);
    int getEntriesForBroadcast(std::vector<SearchEntry> &list);
    void doSearch(std::vector<SearchEntry> &entries, const char *query);
    void flush();
    void close();

private:
    std::map<SearchEntryHash256, time_t> timestamps;
    DB *dbp;

    SearchEntry getByHash(const SearchEntryHash256 &hash);
    void modified();
};

#endif // SEARCHENTRIESDB_H
