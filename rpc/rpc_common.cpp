#include "rpc_common.h"

/*
    PingArgs
*/
PingArgs::PingArgs()
    : address("")
{
    memset(this->id.id, 0, KAD_ID_SZ);
}

PingArgs::PingArgs(const KadId &id_, std::string address_)
    : address(address_), id(id_)
{}

int PingArgs::read(SdsBytesBuf &buf)
{
    address = buf.readString();
    return buf.readBytes(id.id, KAD_ID_SZ) == KAD_ID_SZ;
}

void PingArgs::write(SdsBytesBuf &buf)
{
    buf.writeString(address);
    buf.writeBytes(id.id, KAD_ID_SZ);
}

/*
    FindNodeArgs
*/
FindNodeArgs::FindNodeArgs()
{
    memset(this->id.id, 0, KAD_ID_SZ);
}

FindNodeArgs::FindNodeArgs(const KadId &id_)
    : id(id_)
{}

void FindNodeArgs::read(SdsBytesBuf &buf)
{
    buf.readBytes(id.id, KAD_ID_SZ);
}

void FindNodeArgs::write(SdsBytesBuf &buf)
{
    buf.writeBytes(id.id, KAD_ID_SZ);
}

/*
    FindNodeReply
*/
void FindNodeReply::read(SdsBytesBuf &buf)
{
    int size = buf.readInt32();
    for (int i = 0; i < size; i++) {
        KadId id;
        buf.readBytes(id.id, KAD_ID_SZ);

        std::string addr = buf.readString();
        nearest[id] = addr;
    }
}

void FindNodeReply::write(SdsBytesBuf &buf)
{
    buf.writeInt32(nearest.size());
    for (auto it = nearest.begin(); it != nearest.end(); it++) {
        buf.writeBytes(it->first.id, KAD_ID_SZ);
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

void FindResultsReply::read(SdsBytesBuf &buf)
{
    int size = buf.readInt32();
    for (int i = 0; i < size; i++) {
        SearchEntry se;
        se.read(buf);
        results.push_back(se);
    }
}

void FindResultsReply::write(SdsBytesBuf &buf)
{
    buf.writeInt32(results.size());
    for (auto it = results.begin(); it != results.end(); it++) {
        it->write(buf);
    }
}
