#include <string.h>

#include "utils.h"
#include "macros.h"

#include "searchentriesdb.h"

/*****************************************************************

        SearchEntry

*****************************************************************/

SearchEntry::SearchEntry()
    : SearchEntry("", "", SITE, {})
{}

SearchEntry::SearchEntry(const std::string title, const std::string url, SearchEntryType type, std::map<uint8_t, std::string> properties)
    : title(title), url(url), type(type), properties(properties)
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

uint8_t *SearchEntry::toBytes() const
{
    uint8_t *bytes = new uint8_t[MAX_SEARCH_ENTRY_SZ];
    memset(bytes, 0, MAX_SEARCH_ENTRY_SZ);
    uint8_t *bytesp = bytes;
    int i;
    size_t len;
    for (i = 0; i < 4; i++) {
        memcpy(bytesp, this->metrics[i].id, KAD_ID_SZ);
        bytesp += KAD_ID_SZ;
    }
    len = this->title.length() + 1;
    memcpy(bytesp, this->title.c_str(), len);
    bytesp += len;
    len = this->url.length() + 1;
    memcpy(bytesp, this->url.c_str(), len);
    bytesp += len;
    *bytesp = (uint8_t) this->type;
    bytesp++;
    for (auto it = this->properties.begin(); it != this->properties.end(); it++) {
        len = it->second.length() + 1;
        if (MAX_SEARCH_ENTRY_SZ < ((bytesp - bytes) + len + 1))
            return nullptr;

        *bytesp = (uint8_t) it->first;
        bytesp++;
        memcpy(bytesp, it->second.c_str(), len);
        bytesp += len;
    }
    return bytes;
}

int SearchEntry::fromBytes(uint8_t *buf)
{
    size_t len;
    int i;
    size_t max_len = MAX_SEARCH_ENTRY_SZ - 4*KAD_ID_SZ;

    if (!buf)
        return -1;

    if (max_len < 0)
        return -2;

    for (i = 0; i < 4; i++) {
        memcpy(this->metrics[i].id, buf, KAD_ID_SZ);
        buf += KAD_ID_SZ;
    }

    len = strlen((char*) buf) + 1;
    max_len -= len;
    if (max_len < 0)
        return -2;

    this->title = std::string(buf, buf +len);
    buf += len;

    len = strlen((char*) buf) + 1;
    max_len -= len;
    if (max_len < 0)
        return -2;

    this->url = std::string(buf, buf +len);
    buf += len;

    max_len -= 1;
    if (max_len < 0)
        return -2;

    this->type = (SearchEntryType) *buf;
    buf++;

    uint8_t idx = 0;
    while (max_len >= 0) {
        max_len -= 1;
        if (max_len < 0)
            return -2;

        idx = *buf;
        buf++;
        len = strlen((char*) buf) + 1;
        if (len == 1)
            return 0;

        max_len -= len;
        if (max_len < 0)
            return -2;

        this->properties[idx] = std::string(buf, buf +len);
    }

    return 0;
}

SearchEntryHash256 SearchEntry::getHash() const
{
    return hash;
}

bool SearchEntry::unpack(msgpack11::MsgPack &obj)
{
    if (!obj.is_array())
        return false;

    if (!obj[0].is_string())
        return false;
    memcpy(this->hash.data(), obj[0].string_value().c_str(), SHA256_DIGEST_LENGTH);

    if (!obj[1].is_array())
        return false;
    msgpack11::MsgPack::array d = obj[1].array_items();
    int i;
    for (i = 0; i < 4; i++) {
        if (!d[i].is_string())
            return false;
        memcpy(this->metrics[i].id, d[i].string_value().c_str(), KAD_ID_SZ);
    }

    if (!obj[2].is_string())
        return false;
    this->title = obj[2].string_value();

    if (!obj[3].is_string())
        return false;
    this->url = obj[3].string_value();

    if (!obj[4].is_int())
        return false;
    this->type = (SearchEntryType) obj[4].int_value();

    if (!obj[5].is_object())
        return false;

    msgpack11::MsgPack::object props = obj[5].object_items();
    this->properties.clear();
    for (auto it = props.begin(); it != props.end(); it++) {
        if (!it->first.is_int() || !it->second.is_string())
            return false;
        this->properties[(uint8_t) it->first.int_value()] = it->second.string_value();
    }

    return true;
}

void SearchEntry::pack(msgpack11::MsgPack &obj)
{
    msgpack11::MsgPack::array o;
    o.push_back(std::string(this->hash.data(), this->hash.data() + this->hash.size()));

    msgpack11::MsgPack::array d;
    int i;
    for (int i = 0; i < 4; i++)
        d.push_back(std::string(this->metrics[i].id, this->metrics[i].id + KAD_ID_SZ));

    o.push_back(d);
    o.push_back(this->title);
    o.push_back(this->url);
    o.push_back((int) this->type);

    msgpack11::MsgPack::object props;
    for (auto it = this->properties.begin(); it != this->properties.end(); it++)
        props[(int) it->first] = it->second;

    o.push_back(props);
    obj = o;
}

KadId *SearchEntry::getMetrics() const
{
    return (KadId*) metrics;
}

std::ostream &operator<<(std::ostream &os, const SearchEntry &se)
{
    int i;
    os << "SearchEntry["
       << "hash=";

    STREAM_HEX(os, se.hash, SHA256_DIGEST_LENGTH);

    os << ", title=" << se.title
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

}

/*****************************************************************

        SearchEntryDB

*****************************************************************/

SearchEntriesDB::SearchEntriesDB()
    : timestamps({}), dbp(nullptr)
{}

SearchEntriesDB::~SearchEntriesDB()
{
    this->close();
    delete this->dbp;
}

void SearchEntriesDB::open(const char *db_path)
{
    int ret = 0;
    if ((ret = db_create(&this->dbp, nullptr, 0))) {
        return;
    }
    if ((ret = this->dbp->set_cachesize(this->dbp, 0, 128 * 1024, 0)) != 0) {
        return;
    }
    if ((ret = this->dbp->open(this->dbp, nullptr, db_path, nullptr, DB_HASH, DB_CREATE/* | DB_THREAD*/, 0664)) != 0) {
        return;
    }
    this->modified();

    DBT key, data;
    DBC *dbcp;
    if ((ret = this->dbp->cursor(this->dbp, nullptr, &dbcp, 0)) != 0) {
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

    }
}

void SearchEntriesDB::insertResult(const SearchEntry &se)
{
    if (!this->dbp)
        return;

    this->timestamps[se.getHash()] = time(nullptr);

    uint8_t *bytes = se.toBytes();

    DBT key, data;
    key.data = se.getHash().data();
    key.size = SHA256_DIGEST_LENGTH;
    data.data = bytes;
    data.size = MAX_SEARCH_ENTRY_SZ;
    this->dbp->put(this->dbp, nullptr, &key, &data, 0);

    this->modified();

    delete[] bytes;
}

void SearchEntriesDB::getEntriesForBroadcast(std::vector<SearchEntry> &list)
{
    if (!this->dbp)
        return;

    list.clear();

    time_t now = time(nullptr);
    for (auto it = this->timestamps.begin(); it != this->timestamps.end(); it++) {
        if ((now - it->second) >= UNIX_HOUR) {
            list.push_back(getByHash(it->first));
            it->second = now;
        }
    }

    this->modified();
}

void SearchEntriesDB::doSearch(std::vector<SearchEntry> &entries, const char *query)
{

}

void SearchEntriesDB::flush()
{
    if (!this->dbp)
        return;

    if (this->dbp->sync(this->dbp, 0)) {
    }
}

void SearchEntriesDB::close()
{
    if (!this->dbp)
        return;

    if (this->dbp->close(this->dbp, 0)) {
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
        se.fromBytes((uint8_t*) data.data);
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
