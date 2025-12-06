#include "sdsbytesbuf.h"

#include <endian.h>

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
    this->zero();
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

void SdsBytesBuf::zero()
{
    if (this->buffer) {
        free(this->buffer);
        this->pos = 0;
        this->bufferSize = 0;

        this->buffer = nullptr;
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

void SdsBytesBuf::writeBytes(const char *buffer, size_t bufferSize)
{
    if (buffer && bufferSize)
        this->write(buffer, bufferSize);
}

int SdsBytesBuf::readBytes(char *buffer, size_t bufferSize)
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

    std::string str(cursor, cursor + len);

    this->pos += len + 1;

    return str;
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

void SdsBytesBuf::writeUint16(uint16_t n)
{
    uint16_t le = htole16(n);
    this->write(&le, sizeof(uint16_t));
}

uint16_t SdsBytesBuf::readUint16()
{
    uint16_t le = 0;
    this->read(&le, sizeof(uint16_t));
    return le16toh(le);
}

void SdsBytesBuf::writeUint32(uint32_t n)
{
    uint32_t le = htole32(n);
    this->write(&le, sizeof(uint32_t));
}

uint32_t SdsBytesBuf::readUint32()
{
    uint32_t le = 0;
    this->read(&le, sizeof(uint32_t));
    return le32toh(le);
}

void SdsBytesBuf::writeUint64(uint64_t n)
{
    uint64_t le = htole64(n);
    this->write(&le, sizeof(uint64_t));
}

uint64_t SdsBytesBuf::readUint64()
{
    uint64_t le = 0;
    this->read(&le, sizeof(uint64_t));
    return le64toh(le);
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
