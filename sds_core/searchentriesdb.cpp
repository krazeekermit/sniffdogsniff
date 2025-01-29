#include <string.h>

#include "common/stringutil.h"
#include "common/utils.h"
#include "common/macros.h"
#include "common/logging.h"

#include "searchentriesdb.h"

/*****************************************************************

        SearchEntry

*****************************************************************/

SearchEntry::SearchEntry()
    : SearchEntry("", "", SITE, {})
{}

SearchEntry::SearchEntry(const std::string title, const std::string url, SearchEntryType type, std::map<uint8_t, std::string> properties)
    : simHash(title), title(title), url(url), type(type), properties(properties)
{
    this->hash = {};
    reHash();

    evaluateDistances();
}

SearchEntry::~SearchEntry()
{
}

void SearchEntry::addProperty(uint8_t idx, const std::string value)
{
    this->properties[idx] = value;
}

const std::string SearchEntry::getProperty(uint8_t idx)
{
    return this->properties[idx];
}

void SearchEntry::removeProperty(uint8_t idx)
{
    this->properties.erase(idx);
}

/* Calculate the SHA256 of the URL */
void SearchEntry::reHash()
{    
    SHA256_CTX ctx;
    SHA256_Init(&ctx);
    SHA256_Update(&ctx, this->url.c_str(), this->url.length());
    SHA256_Final(this->hash.data(), &ctx);
}

void SearchEntry::read(SdsBytesBuf &buf)
{
    this->simHash.read(buf);
    this->title = buf.readString();
    this->url = buf.readString();
    this->type = static_cast<SearchEntryType>(buf.readUint8());
    this->properties.clear();
    int propsSize = buf.readInt32();
    for (int i = 0; i < propsSize; i++) {
        uint8_t k = buf.readUint8();
        std::string val = buf.readString();
        this->properties[k] = val;
    }

    this->reHash();
}

void SearchEntry::write(SdsBytesBuf &buf)
{
    this->simHash.write(buf);
    buf.writeString(this->title);
    buf.writeString(this->url);
    buf.writeUint8(static_cast<uint8_t>(this->type));
    buf.writeInt32(this->properties.size());
    for (auto it = this->properties.begin(); it != this->properties.end(); it++) {
        buf.writeUint8(it->first);
        buf.writeString(it->second);
    }
}

SearchEntryHash256 SearchEntry::getHash() const
{
    return hash;
}

SimHash SearchEntry::getSimHash() const
{
    return this->simHash;
}

std::string SearchEntry::getTitle() const
{
    return title;
}

std::string SearchEntry::getUrl() const
{
    return url;
}

std::ostream &operator<<(std::ostream &os, const SearchEntry &se)
{
    int i;
    os << "SearchEntry["
       << "hash=";

    STREAM_HEX(os, se.hash, SHA256_DIGEST_LENGTH);

    os << ", simHash=" << se.simHash
       << ", title=" << se.title
       << ", url=" << se.url
       << ", type=" << (int) se.type
       << ", properties=[";

    for (auto it = se.properties.begin(); it != se.properties.end(); it++) {
        os << (int) it->first <<"=" << it->second << ", ";
    }
    os << "]" << "]";

    return os;
}

void SearchEntry::evaluateDistances()
{
    //SearchEntry::evaluateMetrics(this->metrics, this->title.c_str());
}

/*****************************************************************

        SearchEntryDB

*****************************************************************/

SearchEntriesDB::SearchEntriesDB()
    : timestamps({}), dbp(nullptr)
{}

SearchEntriesDB::~SearchEntriesDB()
{
}

void SearchEntriesDB::open(const char *db_path)
{
    int ret = 0;
    if ((ret = db_create(&this->dbp, nullptr, 0))) {
        logdebug << "unable to open db file " << db_path << ": " << db_strerror(ret);
        return;
    }
    if ((ret = this->dbp->set_cachesize(this->dbp, 0, 128 * 1024, 0)) != 0) {
        logdebug << "unable to set db cache sz " << db_path << ": " << db_strerror(ret);
        return;
    }
    if ((ret = this->dbp->open(this->dbp, nullptr, db_path, nullptr, DB_HASH, DB_CREATE | DB_THREAD, 0664)) != 0) {
        logerr << "unable to open db file " << db_path << ": " << db_strerror(ret);
        return;
    }
    this->modified();

    DBT key, data;
    DBC *dbcp;
    if ((ret = this->dbp->cursor(this->dbp, nullptr, &dbcp, 0)) != 0) {
        logerr << db_strerror(ret);
        return;
    }

    memset(&key, 0, sizeof(key));
    memset(&data, 0, sizeof(data));

    while ((ret = dbcp->get(dbcp, &key, &data, DB_NEXT)) == 0) {
        std::array<uint8_t, SHA256_DIGEST_LENGTH> hash;
        memcpy(hash.data(), key.data, key.size);
        this->timestamps[hash] = time(nullptr);
    }

    if (ret != DB_NOTFOUND) {
        return;
    }

    if ((ret = dbcp->close(dbcp)) != 0) {
        logerr << db_strerror(ret);
    }
}

void SearchEntriesDB::insertResult(SearchEntry &se)
{
    if (!this->dbp)
        return;

    this->timestamps[se.getHash()] = time(nullptr);    

    DBT key, data;
    key.data = se.getHash().data();
    key.size = SHA256_DIGEST_LENGTH;

    SdsBytesBuf buf;
    se.write(buf);

    data.data = buf.bufPtr();
    data.size = buf.size();
    this->dbp->put(this->dbp, nullptr, &key, &data, 0);

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

static int similarTo(SearchEntry &se, std::vector<std::string> &tokens)
{
    // to be implemented...
    int count = 0;
    return count + 1;
}

void SearchEntriesDB::doSearch(std::vector<SearchEntry> &entries, std::string query)
{
    int ret = 0;
    DBT key, data;
    DBC *dbcp;
    if ((ret = this->dbp->cursor(this->dbp, nullptr, &dbcp, 0)) != 0) {
        logerr << db_strerror(ret);
        return;
    }

    memset(&key, 0, sizeof(key));
    memset(&data, 0, sizeof(data));

    std::vector<std::string> queryTokens = tokenize(query, " \n\r", ".:,;()[]{}#@");
    SimHash queryHash(queryTokens);

    std::vector<SearchEntry> candidates;
    while ((ret = dbcp->get(dbcp, &key, &data, DB_NEXT)) == 0) {
        SearchEntry se;
        SdsBytesBuf buf(data.data, data.size);
        se.read(buf);

        if (se.getSimHash().distance(queryHash) < 48) {
            candidates.push_back(se);
        }
    }

    if (ret != DB_NOTFOUND) {
        return;
    }

    if ((ret = dbcp->close(dbcp)) != 0) {
        logerr << db_strerror(ret);
    }

    for (int i = 0; i < candidates.size(); i++) {
        SearchEntry candidate = candidates[i];
        if (similarTo(candidate, queryTokens)) {
            entries.push_back(candidate);
        }
    }
}

void SearchEntriesDB::flush()
{
    if (!this->dbp)
        return;

    int dberr = this->dbp->sync(this->dbp, 0);
    if (dberr) {
        logerr << "SearchEntriesDB error: " << db_strerror(dberr);
    }
}

void SearchEntriesDB::close()
{
    if (!this->dbp)
        return;

    int dberr = this->dbp->close(this->dbp, 0);
    if (dberr) {
        logerr << "SearchEntriesDB error: " << db_strerror(dberr);
    }
}

SearchEntry SearchEntriesDB::getByHash(const SearchEntryHash256 &hash)
{
    DBT key, data;
    SearchEntry se;
    memset(&key, 0, sizeof(key));
    memset(&data, 0, sizeof(data));

    key.data = (unsigned char*) hash.data();
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
    time_t now = time(nullptr);

    for (auto it = this->timestamps.begin(); it != this->timestamps.end();) {
        if ((now - it->second) >= UNIX_DAY) {
            key.data = (unsigned char*) it->first.data();
            this->dbp->del(this->dbp, nullptr, &key, 0);
            this->timestamps.erase(it);
        } else {
            it++;
        }
    }
}
