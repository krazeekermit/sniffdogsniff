#ifndef SDSBYTESBUF_H
#define SDSBYTESBUF_H

#include <cstdint>
#include <iostream>

class SdsBytesBuf
{
public:
    SdsBytesBuf();
    SdsBytesBuf(void *inp, size_t size);
    ~SdsBytesBuf();

    void allocate(size_t size);
    void resize(size_t size);
    void rewind();
    size_t size() const;
    uint8_t *bufPtr() const;

    void writeBytes(const uint8_t *buffer, size_t bufferSize);
    int readBytes(uint8_t *buffer, size_t bufferSize);

    void writeString(const char *str);
    void writeString(std::string &str);
    std::string readString();

    void writeInt8(int8_t n);
    int8_t readInt8();

    void writeUint8(uint8_t n);
    uint8_t readUint8();

    void writeInt16(int16_t n);
    int16_t readInt16();

    void writeUint16(uint16_t n);
    uint16_t readUint16();

    void writeInt32(int32_t n);
    int32_t readInt32();

    void writeUint32(uint32_t n);
    uint32_t readUint32();

    void writeInt64(int64_t n);
    int64_t readInt64();

    void writeUint64(uint64_t n);
    uint64_t readUint64();

    void writeBool(bool b);
    bool readBool();

private:
    size_t bufferSize;
    size_t capacity;
    size_t pos;
    uint8_t *buffer;

    uint8_t *cur() const;

    int read(void *out, size_t size);
    int write(const void *in, size_t size);
};

#endif // SDSBYTESBUF_H
