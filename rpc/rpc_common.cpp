#include "rpc_common.h"

/*
    ArgsBase
*/

ArgsBase::ArgsBase(const KadId &id_, std::string address_)
    : callerId(id_), callerAddress(address_)
{
}

void ArgsBase::read(SdsBytesBuf &buf)
{
    callerAddress = buf.readString();
    buf.readBytes(callerId.id, KAD_ID_LENGTH);
}

void ArgsBase::write(SdsBytesBuf &buf)
{
    buf.writeString(callerAddress);
    buf.writeBytes(callerId.id, KAD_ID_LENGTH);
}

/*
    PingArgs
*/
PingArgs::PingArgs(const KadId &callerId_, std::string callerAddress_)
    : ArgsBase(callerId_, callerAddress_)
{
}

/*
    FindNodeArgs
*/
FindNodeArgs::FindNodeArgs(const KadId &callerId_, std::string callerAddress_, const KadId &targetId_)
    : ArgsBase(callerId_, callerAddress_), targetId(targetId_)
{
}

void FindNodeArgs::read(SdsBytesBuf &buf)
{
    ArgsBase::read(buf);
    buf.readBytes(targetId.id, KAD_ID_LENGTH);
}

void FindNodeArgs::write(SdsBytesBuf &buf)
{
    ArgsBase::write(buf);
    buf.writeBytes(targetId.id, KAD_ID_LENGTH);
}

/*
    FindNodeReply
*/
void FindNodeReply::read(SdsBytesBuf &buf)
{
    int size = buf.readInt32();
    for (int i = 0; i < size; i++) {
        KadId id;
        buf.readBytes(id.id, KAD_ID_LENGTH);

        std::string addr = buf.readString();
        nearest[id] = addr;
    }
}

void FindNodeReply::write(SdsBytesBuf &buf)
{
    buf.writeInt32(nearest.size());
    for (auto it = nearest.begin(); it != nearest.end(); it++) {
        buf.writeBytes(it->first.id, KAD_ID_LENGTH);
        buf.writeString(it->second);
    }
}

/*
    StoreResultArgs
*/
StoreResultArgs::StoreResultArgs(const KadId &callerId_, std::string callerAddress_, SearchEntry se_)
    : ArgsBase(callerId_, callerAddress_), se(se_)
{}

void StoreResultArgs::read(SdsBytesBuf &buf)
{
    ArgsBase::read(buf);
    se.read(buf);
}

void StoreResultArgs::write(SdsBytesBuf &buf)
{
    ArgsBase::write(buf);
    se.write(buf);
}

/*
    FindResults
*/
FindResultsArgs::FindResultsArgs(const KadId &callerId_, std::string callerAddress_, std::string query_)
    : ArgsBase(callerId_, callerAddress_), query(query_)
{
}

void FindResultsArgs::read(SdsBytesBuf &buf)
{
    ArgsBase::read(buf);
    query = buf.readString();
}

void FindResultsArgs::write(SdsBytesBuf &buf)
{
    ArgsBase::write(buf);
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
