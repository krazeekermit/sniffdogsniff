#ifndef SEARCHENTRIESDB_H
#define SEARCHENTRIESDB_H

#include "simhash.h"

#include "searchentry.h"

#include <map>
#include <cstdint>
#include <iostream>
#include <ctime>
#include <vector>
#include <array>
#include <openssl/sha.h>
#include <db.h>

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
    std::map<SearchEntry::Hash, time_t> timestamps;
    DB *dbp;

    SearchEntry getByHash(const SearchEntry::Hash &hash);
    void modified();
};

#endif // SEARCHENTRIESDB_H
