#include "p2p_common.h"

/*
    PingArgs
*/
PingArgs::PingArgs(const KadId &id_, std::string address_)
    : id(id_), address(address_)
{}

void PingArgs::read(SdsBytesBuf &buf)
{
    address = buf.readString();
    buf.readBytes(id.id, KAD_ID_LENGTH);
}

void PingArgs::write(SdsBytesBuf &buf)
{
    buf.writeString(address);
    buf.writeBytes(id.id, KAD_ID_LENGTH);
}

/*
    FindNodeArgs
*/
FindNodeArgs::FindNodeArgs(const KadId &targetId_)
    : targetId(targetId_)
{}

void FindNodeArgs::read(SdsBytesBuf &buf)
{
    buf.readBytes(targetId.id, KAD_ID_LENGTH);
}

void FindNodeArgs::write(SdsBytesBuf &buf)
{
    buf.writeBytes(targetId.id, KAD_ID_LENGTH);
}

/*
    FindNodeReply
*/
void FindNodeReply::read(SdsBytesBuf &buf)
{
    unsigned int size = buf.readUint32();
    for (unsigned int i = 0; i < size; i++) {
        KadId id;
        buf.readBytes(id.id, KAD_ID_LENGTH);

        std::string addr = buf.readString();
        nearest[id] = addr;
    }
}

void FindNodeReply::write(SdsBytesBuf &buf)
{
    buf.writeUint32(nearest.size());
    for (auto it = nearest.begin(); it != nearest.end(); it++) {
        buf.writeBytes(it->first.id, KAD_ID_LENGTH);
        buf.writeString(it->second);
    }
}

/*
    StoreResultArgs
*/
StoreResultArgs::StoreResultArgs(SearchEntry se_)
    : se(se_)
{}

void StoreResultArgs::read(SdsBytesBuf &buf)
{
    se.read(buf);
}

void StoreResultArgs::write(SdsBytesBuf &buf)
{
    se.write(buf);
}

/*
    FindResults
*/
FindResultsArgs::FindResultsArgs(std::string query_)
    : query(query_)
{
}

void FindResultsArgs::read(SdsBytesBuf &buf)
{
    query = buf.readString();
}

void FindResultsArgs::write(SdsBytesBuf &buf)
{
    buf.writeString(query);
}

bool FindResultsReply::hasResults()
{
    return this->results.size() > 0 && this->nearest.empty();
}

void FindResultsReply::read(SdsBytesBuf &buf)
{
    FindNodeReply::read(buf);

    unsigned int size = buf.readUint32();
    for (unsigned int i = 0; i < size; i++) {
        SearchEntry se;
        se.read(buf);
        results.push_back(se);
    }
}

void FindResultsReply::write(SdsBytesBuf &buf)
{
    FindNodeReply::write(buf);

    buf.writeUint32(results.size());
    for (auto it = results.begin(); it != results.end(); it++) {
        it->write(buf);
    }
}
