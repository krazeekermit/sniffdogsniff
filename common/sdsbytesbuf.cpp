#include "sdsbytesbuf.h"

#include <cstring>
#include <cstdlib>

SdsBytesBuf::SdsBytesBuf()
    : buffer(nullptr), pos(0), capacity(0), bufferSize(0)
{
}

SdsBytesBuf::SdsBytesBuf(void *inp, size_t size)
    : SdsBytesBuf()
{
    this->write(inp, size);
}

SdsBytesBuf::~SdsBytesBuf()
{
    if (this->buffer)
        free(this->buffer);
}

void SdsBytesBuf::allocate(size_t size)
{
    this->capacity = size;
    this->buffer = static_cast<uint8_t*>(malloc(this->capacity * sizeof(uint8_t)));
}

void SdsBytesBuf::resize(size_t size)
{
    if (this->buffer) {
        if (this->capacity < size) {
            while (this->capacity < size)
                this->capacity *= 2;

            this->buffer = static_cast<uint8_t*>(realloc(this->buffer, this->capacity * sizeof(uint8_t)));
        }
    } else {
        this->allocate(size);
    }
}

void SdsBytesBuf::rewind()
{
    this->pos = 0;
}

size_t SdsBytesBuf::size() const
{
    return this->bufferSize;
}

uint8_t *SdsBytesBuf::bufPtr() const
{
    return this->buffer;
}

void SdsBytesBuf::writeBytes(const uint8_t *buffer, size_t bufferSize)
{
    if (buffer && bufferSize)
        this->write(buffer, bufferSize);
}

int SdsBytesBuf::readBytes(uint8_t *buffer, size_t bufferSize)
{
    if (buffer && bufferSize)
        return this->read(buffer, bufferSize);

    return 0;
}

void SdsBytesBuf::writeString(const char *str)
{
    std::string s(str);
    this->writeString(s);
}

void SdsBytesBuf::writeString(std::string &str)
{
    this->write(str.c_str(), str.length()+1);
}

std::string SdsBytesBuf::readString()
{
    size_t len = 0;
    const uint8_t *cursor = this->cur();
    while (len < (this->bufferSize - this->pos)) {
        len++;
        if (cursor[len] == '\0')
            break;
    }
    len++;

    std::string str(cursor, cursor + len);
    this->pos += len;

    if (len == (this->bufferSize - this->pos))
        str += '\0';

    return str;
}

void SdsBytesBuf::writeInt8(int8_t n)
{
    this->write(&n, sizeof(int8_t));
}

int8_t SdsBytesBuf::readInt8()
{
    int8_t n = 0;
    this->read(&n, sizeof(int8_t));
    return n;
}

void SdsBytesBuf::writeUint8(uint8_t n)
{
    this->write(&n, sizeof(uint8_t));
}

uint8_t SdsBytesBuf::readUint8()
{
    uint8_t n = 0;
    this->read(&n, sizeof(uint8_t));
    return n;
}

void SdsBytesBuf::writeInt16(int16_t n)
{
    this->write(&n, sizeof(int16_t));
}

int16_t SdsBytesBuf::readInt16()
{
    int16_t n = 0;
    this->read(&n, sizeof(int16_t));
    return n;
}

void SdsBytesBuf::writeUint16(uint16_t n)
{
    this->write(&n, sizeof(uint16_t));
}

uint16_t SdsBytesBuf::readUint16()
{
    uint16_t n = 0;
    this->read(&n, sizeof(uint16_t));
    return n;
}

void SdsBytesBuf::writeInt32(int32_t n)
{
    this->write(&n, sizeof(int32_t));
}

int32_t SdsBytesBuf::readInt32()
{
    int32_t n = 0;
    this->read(&n, sizeof(int32_t));
    return n;
}

void SdsBytesBuf::writeUint32(uint32_t n)
{
    this->write(&n, sizeof(uint32_t));
}

uint32_t SdsBytesBuf::readUint32()
{
    uint32_t n = 0;
    this->read(&n, sizeof(uint32_t));
    return n;
}

void SdsBytesBuf::writeInt64(int64_t n)
{
    this->write(&n, sizeof(int64_t));
}

int64_t SdsBytesBuf::readInt64()
{
    int64_t n = 0;
    this->read(&n, sizeof(int64_t));
    return n;
}

void SdsBytesBuf::writeUint64(uint64_t n)
{
    this->write(&n, sizeof(uint64_t));
}

uint64_t SdsBytesBuf::readUint64()
{
    uint64_t n = 0;
    this->read(&n, sizeof(uint64_t));
    return n;
}

void SdsBytesBuf::writeBool(bool b)
{
    this->writeUint8(b == true ? 1 : 0);
}

bool SdsBytesBuf::readBool()
{
    return this->readUint8() == 1;
}

//********************* PRIVATE **********************+

uint8_t *SdsBytesBuf::cur() const
{
    return this->buffer + this->pos;
}

int SdsBytesBuf::read(void *out, size_t size)
{
    if (this->bufferSize - this->pos >= size) {
        memcpy(out, this->cur(), size);
        this->pos += size;
        return size;
    }

    return 0;
}

int SdsBytesBuf::write(const void *in, size_t size)
{
    resize(this->bufferSize + size);

    memcpy(this->cur(), in, size);
    this->pos += size;
    this->bufferSize += size;
    return size;
}
