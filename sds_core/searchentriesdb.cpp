#include <string.h>

#include "common/stringutil.h"
#include "common/utils.h"
#include "common/macros.h"
#include "common/loguru.hpp"

#include "searchentriesdb.h"

#define MINIMUM_SIMHASH_DISTANCE 48

SearchEntriesDB::SearchEntriesDB()
    : timestamps({}), dbp(nullptr)
{}

SearchEntriesDB::~SearchEntriesDB()
{
    this->dbp = nullptr;
}

void SearchEntriesDB::open(const char *db_path)
{
    int ret = 0;
    if ((ret = db_create(&this->dbp, nullptr, 0))) {
        LOG_F(1, "unable to open db file %s: %s", db_path, db_strerror(ret));
        return;
    }
    if ((ret = this->dbp->set_cachesize(this->dbp, 0, 128 * 1024, 0)) != 0) {
        LOG_F(1, "unable to set db cache sz %s: %s", db_path, db_strerror(ret));
        return;
    }
    if ((ret = this->dbp->open(this->dbp, nullptr, db_path, nullptr, DB_HASH, DB_CREATE | DB_THREAD, 0664)) != 0) {
        LOG_F(ERROR, "unable to open db file %s: %s", db_path, db_strerror(ret));
        return;
    }
    this->modified();

    DBT key, data;
    DBC *dbcp;
    if ((ret = this->dbp->cursor(this->dbp, nullptr, &dbcp, 0)) != 0) {
        LOG_F(ERROR, db_strerror(ret));
        return;
    }

    memset(&key, 0, sizeof(key));
    memset(&data, 0, sizeof(data));

    while ((ret = dbcp->get(dbcp, &key, &data, DB_NEXT)) == 0) {
        this->timestamps[SearchEntry::Hash((uint8_t*) key.data)] = time(nullptr);
    }

    if (ret != DB_NOTFOUND) {
        return;
    }

    if ((ret = dbcp->close(dbcp)) != 0) {
        LOG_F(ERROR, db_strerror(ret));
    }
}

void SearchEntriesDB::insertResult(SearchEntry &se)
{
    if (!this->dbp)
        return;

    if (!se.hasValidSize()) {
        throw std::runtime_error(
            "search entry exceeds size limit of " + std::to_string(MAX_SEARCH_ENTRY_SIZE) + " bytes"
            );
    }

    this->timestamps[se.getHash()] = time(nullptr);    

    DBT key, data;
    memset(&key, 0, sizeof(key));
    memset(&data, 0, sizeof(data));

    key.data = se.getHash().hash;
    key.size = SHA256_DIGEST_LENGTH;

    SdsBytesBuf buf;
    se.write(buf);

    data.data = buf.bufPtr();
    data.size = buf.size();
    int dberr = this->dbp->put(this->dbp, nullptr, &key, &data, 0);
    if (dberr) {
        throw std::runtime_error("insert result fail database error: " + std::string(db_strerror(dberr)));
    }

    this->modified();
}

int SearchEntriesDB::getEntriesForBroadcast(std::vector<SearchEntry> &list)
{
    if (!this->dbp)
        return 0;

    time_t now = time(nullptr);
    for (auto it = this->timestamps.begin(); it != this->timestamps.end(); it++) {
        if ((now - it->second) >= UNIX_HOUR) {
            list.push_back(getByHash(it->first));
            it->second = now;
        }
    }

    this->modified();

    return list.size();
}

void SearchEntriesDB::doSearch(std::vector<SearchEntry> &entries, std::string query)
{
    int ret = 0;
    DBT key, data;
    DBC *dbcp;
    if ((ret = this->dbp->cursor(this->dbp, nullptr, &dbcp, 0)) != 0) {
        LOG_F(ERROR, db_strerror(ret));
        return;
    }

    memset(&key, 0, sizeof(key));
    memset(&data, 0, sizeof(data));

    std::vector<std::string> queryTokens = tokenize(query, " \n\r", ".:,;()[]{}#@");
    SimHash queryHash(queryTokens);

    while ((ret = dbcp->get(dbcp, &key, &data, DB_NEXT)) == 0) {
        SearchEntry se;
        SdsBytesBuf buf(data.data, data.size);
        se.read(buf);

        if (se.getSimHash().distance(queryHash) < MINIMUM_SIMHASH_DISTANCE && se.matchesQuery(queryTokens)) {
            entries.push_back(se);
        }
    }

    if (ret != DB_NOTFOUND) {
        return;
    }

    if ((ret = dbcp->close(dbcp)) != 0) {
        LOG_F(ERROR, db_strerror(ret));
    }
}

void SearchEntriesDB::flush()
{
    if (!this->dbp)
        return;

    int dberr = this->dbp->sync(this->dbp, 0);
    if (dberr) {
        LOG_F(ERROR, "SearchEntriesDB error: %s", db_strerror(dberr));
    }
}

void SearchEntriesDB::close()
{
    if (!this->dbp)
        return;

    int dberr = this->dbp->close(this->dbp, 0);
    if (dberr) {
        LOG_F(ERROR, "SearchEntriesDB error: %s", db_strerror(dberr));
    }
}

SearchEntry SearchEntriesDB::getByHash(const SearchEntry::Hash &hash)
{
    DBT key, data;
    SearchEntry se;
    memset(&key, 0, sizeof(key));
    memset(&data, 0, sizeof(data));

    key.data = (void*) hash.hash;
    key.size = SHA256_DIGEST_LENGTH;

    int ret = dbp->get(dbp, nullptr, &key, &data, 0);
    if (ret == 0) {
        SdsBytesBuf buf(data.data, data.size);
        se.read(buf);
    }
    return se;
}

void SearchEntriesDB::modified()
{
    DBT key;
    memset(&key, 0, sizeof(key));
    key.size = SHA256_DIGEST_LENGTH;

    time_t now = time(nullptr);

    for (auto it = this->timestamps.begin(); it != this->timestamps.end();) {
        if ((now - it->second) >= UNIX_DAY) {
            key.data = (void*) it->first.hash;
            this->dbp->del(this->dbp, nullptr, &key, 0);
            this->timestamps.erase(it);
        } else {
            it++;
        }
    }
}
