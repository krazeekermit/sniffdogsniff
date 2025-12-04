#include "searchentry.h"

#include "common/sdsbytesbuf.h"
#include "common/stringutil.h"

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
    unsigned int propsSize = buf.readUint32();
    for (unsigned int i = 0; i < propsSize; i++) {
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
