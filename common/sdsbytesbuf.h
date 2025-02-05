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

    void writeUint8(uint8_t n);
    uint8_t readUint8();

    void writeUint16(uint16_t n);
    uint16_t readUint16();

    void writeUint32(uint32_t n);
    uint32_t readUint32();

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
