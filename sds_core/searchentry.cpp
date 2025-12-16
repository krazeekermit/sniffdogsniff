#include "searchentry.h"

#include "common/sdsbytesbuf.h"
#include "common/stringutil.h"

#include "net/netutil.h"

/*
 * SearchEntry::Hash
 */
SearchEntry::Hash::Hash()
{
    memset(this->hash, 0, SHA256_DIGEST_LENGTH);
}

SearchEntry::Hash::Hash(uint8_t *_data)
    : Hash()
{
    if (_data) memcpy(this->hash, _data, SHA256_DIGEST_LENGTH);
}

bool SearchEntry::Hash::operator<(const Hash &hash2) const
{
    return *((uint64_t*) this->hash) < *((uint64_t*) hash2.hash);
}

/*
 * SearchEntry
 */
SearchEntry::SearchEntry()
    : SearchEntry("", "", SITE, {})
{}

SearchEntry::SearchEntry(const std::string title, const std::string url, SearchEntry::Type type, std::map<uint8_t, std::string> properties)
    : simHash(title), title(title), url(url), type(type), properties(properties)
{
    reHash();
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
    SHA256((unsigned char*) this->title.data(), this->title.length(), this->hash.hash);
}

void SearchEntry::read(SdsBytesBuf &buf)
{
    this->simHash.read(buf);
    this->title = buf.readString();
    this->url = buf.readString();
    this->type = static_cast<SearchEntry::Type>(buf.readUint8());
    this->properties.clear();
    unsigned int propsSize = buf.readUint32();
    for (unsigned int i = 0; i < propsSize; i++) {
        uint8_t k = buf.readUint8();
        this->properties[k] = buf.readString();
    }

    this->reHash();
}

void SearchEntry::write(SdsBytesBuf &buf)
{
    this->simHash.write(buf);
    buf.writeString(this->title);
    buf.writeString(this->url);
    buf.writeUint8(static_cast<uint8_t>(this->type));
    buf.writeUint32(this->properties.size());
    for (auto it = this->properties.begin(); it != this->properties.end(); it++) {
        buf.writeUint8(it->first);
        buf.writeString(it->second);
    }
}

bool SearchEntry::matchesQuery(std::vector<std::string> tokens)
{
    int nmatches = 0;
    std::string lowerTitle = toLower(this->title);
    for (auto it = tokens.begin(); it != tokens.end(); it++) {
        if (lowerTitle.find(*it) != std::string::npos) {
            nmatches++;
        }
    }

    for (auto it = tokens.begin(); it != tokens.end(); it++) {
        if (this->url.find(*it) != std::string::npos) {
            nmatches++;
        }
    }

    for (auto pit = this->properties.begin(); pit != this->properties.end(); pit++) {
        std::string pLower = toLower(pit->second);
        for (auto it = tokens.begin(); it != tokens.end(); it++) {
            if (pLower.find(*it) != std::string::npos) {
                nmatches++;
            }
        }
    }

    if (tokens.size() > 1 && split(this->title, " ").size() > 1)
        return nmatches >= 2;

    return nmatches;
}

SearchEntry::Hash SearchEntry::getHash() const
{
    return hash;
}

SimHash SearchEntry::getSimHash() const
{
    return this->simHash;
}

SearchEntry::Type SearchEntry::getType() const
{
    return type;
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
    char hexString[65];
    bytes_to_hex_string(hexString, se.hash.hash, SHA256_DIGEST_LENGTH);

    os << "SearchEntry["
       << "hash=" << hexString
       << ", simHash=" << se.simHash
       << ", title=" << se.title
       << ", url=" << se.url;

    switch (se.type) {
    case SearchEntry::Type::SITE:
        os << ", type=SITE";
        break;
    case SearchEntry::Type::IMAGE:
        os << ", type=IMAGE";
        break;
    case SearchEntry::Type::VIDEO:
        os << ", type=VIDEO";
        break;
    default:
        break;
    }

    os << ", properties=[";
    for (auto it = se.properties.begin(); it != se.properties.end(); it++) {
        os << (int) it->first <<"=" << it->second << ", ";
    }
    os << "]" << "]";

    return os;
}
