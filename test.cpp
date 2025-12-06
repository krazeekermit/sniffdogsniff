#include "common/stringutil.h"
#include "kademlia/kadbucket.h"
#include "kademlia/kadnode.h"
#include "sds_core/simhash.h"
#include "sds_core/searchentry.h"

#include <gtest/gtest.h>

/*
    common tests
*/

TEST(test_common, test_str_split)
{
    std::string str = "cat dog elephant mice";
    auto vec = split(str, " ");
    ASSERT_EQ(vec.size(), 4);

    ASSERT_EQ(vec[0], "cat");
    ASSERT_EQ(vec[1], "dog");
    ASSERT_EQ(vec[2], "elephant");
    ASSERT_EQ(vec[3], "mice");
}

TEST(test_common, test_str_trim_defult_cutset)
{
    std::string str = "\r\ndog \nelephant mice\n\n\r";
    auto trimmed = trim(str);

    ASSERT_EQ(trimmed, "dog \nelephant mice");
}

TEST(test_common, test_str_trim_custom_cutset)
{
    std::string str = "       is the elphant in the room?    \n";
    auto trimmed = trim(str, " \n");

    ASSERT_EQ(trimmed, "is the elphant in the room?");
}

TEST(test_common, test_tokenize)
{
    std::string str = "\n       the    mice   is  in the fridge    \n\r";
    auto toks = tokenize(str);

    ASSERT_EQ(toks.size(), 6);

    ASSERT_EQ(toks[0], "the");
    ASSERT_EQ(toks[1], "mice");
    ASSERT_EQ(toks[2], "is");
    ASSERT_EQ(toks[3], "in");
    ASSERT_EQ(toks[4], "the");
    ASSERT_EQ(toks[5], "fridge");
}

TEST(test_common, test_tokenize_custom_cutset)
{
    std::string str = "\n       (the)    (mouse.mice)   is  [in] t.h.e .freeze@fridge    \n\r";
    auto toks = tokenize(str, " \r\n", "@.[](){}");

    ASSERT_EQ(toks.size(), 6);

    ASSERT_EQ(toks[0], "the");
    ASSERT_EQ(toks[1], "mouse.mice");
    ASSERT_EQ(toks[2], "is");
    ASSERT_EQ(toks[3], "in");
    ASSERT_EQ(toks[4], "t.h.e");
    ASSERT_EQ(toks[5], "freeze@fridge");
}

/*
    kademlia tests
*/

TEST(test_kademlia, test_kadid_height)
{
    const uint8_t idBytes1[] = {0x87, 0x37, 0xfa, 0x6d, 0x7b, 0x6c, 0xf5, 0x6b, 0xa5, 0x1b, 0x26, 0xe5, 0x00, 0x16, 0x81, 0x91};
    KadId id1(idBytes1);

    ASSERT_EQ(id1.height(), 127);

    const uint8_t idBytes2[] = {0x78, 0xa5, 0x76, 0x64, 0x29, 0x66, 0x0f, 0x3b, 0x81, 0x6d, 0xb5, 0xba, 0xde, 0x87, 0x5d, 0x0c};
    KadId id2(idBytes2);

    ASSERT_EQ(id2.height(), 123);

    const uint8_t idBytes3[] = {0x00, 0xaa, 0x00, 0xbb, 0x22, 0x64, 0xcc, 0x3c, 0x8a, 0x4d, 0x2f, 0x9e, 0xb4, 0x81, 0x49, 0x1c};
    KadId id3(idBytes3);

    ASSERT_EQ(id3.height(), 124);

    const uint8_t idBytes4[] = {0x01, 0xa5, 0x30, 0x12, 0x44, 0x00, 0xce, 0xcc, 0xaa, 0xdd, 0xff, 0xee, 0xcc, 0x8d, 0x43, 0x06};
    KadId id4(idBytes4);

    ASSERT_EQ(id4.height(), 122);

    const uint8_t idBytes5[] = {0x29, 0x16, 0x91, 0xe5, 0x24, 0x6e, 0xb2, 0x51, 0x2a, 0xf5, 0x6d, 0x00, 0x00, 0x00, 0x00, 0x00};
    KadId id5(idBytes5);

    ASSERT_EQ(id5.height(), 86);
}

TEST(test_kademlia, test_kadid_xor)
{
    const uint8_t idBytes1[] = {0xb2, 0x3b, 0x30, 0x53, 0xd6, 0x82, 0x07, 0xa3, 0x96, 0x36, 0x74, 0x82, 0xef, 0xc6, 0xcc, 0x75};
    KadId id1(idBytes1);

    const uint8_t idBytes2[] = {0xb4, 0x60, 0x23, 0x02, 0x91, 0xa6, 0x05, 0x32, 0xa1, 0xed, 0xbb, 0x04, 0xbb, 0xb5, 0xde, 0xbe};
    KadId id2(idBytes2);

    KadId xor12 = id1 - id2;
    ASSERT_EQ(xor12.id[0], 0x06);
    ASSERT_EQ(xor12.id[1], 0x5b);
    ASSERT_EQ(xor12.id[2], 0x13);
    ASSERT_EQ(xor12.id[3], 0x51);
    ASSERT_EQ(xor12.id[4], 0x47);
    ASSERT_EQ(xor12.id[5], 0x24);
    ASSERT_EQ(xor12.id[6], 0x02);
    ASSERT_EQ(xor12.id[7], 0x91);
    ASSERT_EQ(xor12.id[8], 0x37);
    ASSERT_EQ(xor12.id[9], 0xdb);
    ASSERT_EQ(xor12.id[10], 0xcf);
    ASSERT_EQ(xor12.id[11], 0x86);
    ASSERT_EQ(xor12.id[12], 0x54);
    ASSERT_EQ(xor12.id[13], 0x73);
    ASSERT_EQ(xor12.id[14], 0x12);
    ASSERT_EQ(xor12.id[15], 0xcb);
}

TEST(test_kademlia, test_knode_create1)
{
    KadNode kn("sniffdogsniff.net:4225");

    ASSERT_EQ(kn.getAddress(), "sniffdogsniff.net:4225");

    const KadId id = kn.getId();
    ASSERT_EQ(id.id[0], 0xed);
    ASSERT_EQ(id.id[1], 0x12);
    ASSERT_EQ(id.id[2], 0x68);
    ASSERT_EQ(id.id[3], 0xfb);
    ASSERT_EQ(id.id[4], 0x6b);
    ASSERT_EQ(id.id[5], 0x2c);
    ASSERT_EQ(id.id[6], 0xaf);
    ASSERT_EQ(id.id[7], 0x81);
    ASSERT_EQ(id.id[8], 0x82);
    ASSERT_EQ(id.id[9], 0x9d);
    ASSERT_EQ(id.id[10], 0x52);
    ASSERT_EQ(id.id[11], 0x77);
    ASSERT_EQ(id.id[12], 0x1c);
    ASSERT_EQ(id.id[13], 0x97);
    ASSERT_EQ(id.id[14], 0x34);
    ASSERT_EQ(id.id[15], 0xa3);
}

TEST(test_kademlia, test_kbucket_addnodes)
{
    KadBucket bucket(1);

    int i;
    for (i = 0; i < 25; i++) {
        KadNode kn(KadId::randomId(), "");
        bucket.pushNode(kn);
    }

    ASSERT_EQ(bucket.getNodesCount(), 20);
    ASSERT_EQ(bucket.getReplacementCount(), 5);
}

TEST(test_kademlia, test_kbucket_stales)
{
    KadBucket bucket(1);

    const uint8_t idBytes1[] = {0x87, 0x37, 0xfa, 0x6d, 0x7b, 0x6c, 0xf5, 0x6b, 0xa5, 0x1b, 0x26, 0xe5, 0x00, 0x16, 0x81, 0x91};
    KadId id1(idBytes1);
    KadNode kn1(id1, "a");
    bucket.pushNode(kn1);

    const uint8_t idBytes2[] = {0x78, 0xa5, 0x76, 0x64, 0x29, 0x66, 0x0f, 0x3b, 0x81, 0x6d, 0xb5, 0xba, 0xde, 0x87, 0x5d, 0x0c};
    KadId id2(idBytes2);
    KadNode kn2(id2, "b");
    bucket.pushNode(kn2);

    const uint8_t idBytes3[] = {0x00, 0xaa, 0x00, 0xbb, 0x22, 0x64, 0xcc, 0x3c, 0x8a, 0x4d, 0x2f, 0x9e, 0xb4, 0x81, 0x49, 0x1c};
    KadId id3(idBytes3);
    KadNode kn3(id3, "c");
    bucket.pushNode(kn3);

    const uint8_t idBytes4[] = {0x01, 0xa5, 0x30, 0x12, 0x44, 0x00, 0xce, 0xcc, 0xaa, 0xdd, 0xff, 0xee, 0xcc, 0x8d, 0x43, 0x06};
    KadId id4(idBytes4);
    KadNode kn4(id4, "d");
    bucket.pushNode(kn4);

    const uint8_t idBytes5[] = {0x29, 0x16, 0x91, 0xe5, 0x24, 0x6e, 0xb2, 0x51, 0x2a, 0xf5, 0x6d, 0x00, 0x00, 0x00, 0x00, 0x00};
    KadId id5(idBytes5);
    KadNode kn5(id5, "e");
    bucket.pushNode(kn5);

    bucket.removeNode(kn2);
    bucket.removeNode(kn2);

    bucket.removeNode(kn3);
    bucket.removeNode(kn3);
    bucket.removeNode(kn3);

    bucket.removeNode(kn5);
    bucket.removeNode(kn5);
    bucket.removeNode(kn5);
    bucket.removeNode(kn5);

    ASSERT_EQ(bucket.getNode(id1).getStales(), 0);
    ASSERT_EQ(bucket.getNode(id2).getStales(), 2);
    ASSERT_EQ(bucket.getNode(id3).getStales(), 3);
    ASSERT_EQ(bucket.getNode(id4).getStales(), 0);
    ASSERT_EQ(bucket.getNode(id5).getStales(), 4);
}

TEST(test_simhash, test_generate)
{
    std::vector<std::string> wlist = {"amused", "anaerobic", "anagram", "anatomist", "excretory" ,"excursion"};
    SimHash sm1(wlist);

    const KadId id = sm1.getId();
    ASSERT_EQ(id.id[0], 0x04);
    ASSERT_EQ(id.id[1], 0x2b);
    ASSERT_EQ(id.id[2], 0x40);
    ASSERT_EQ(id.id[3], 0x0a);
    ASSERT_EQ(id.id[4], 0x4f);
    ASSERT_EQ(id.id[5], 0xd9);
    ASSERT_EQ(id.id[6], 0x83);
    ASSERT_EQ(id.id[7], 0x43);
    ASSERT_EQ(id.id[8], 0xb8);
    ASSERT_EQ(id.id[9], 0x29);
    ASSERT_EQ(id.id[10], 0x81);
    ASSERT_EQ(id.id[11], 0x70);
    ASSERT_EQ(id.id[12], 0x49);
    ASSERT_EQ(id.id[13], 0x80);
    ASSERT_EQ(id.id[14], 0x38);
    ASSERT_EQ(id.id[15], 0xd2);
}

TEST(test_search_entry, test_read_write)
{
    SearchEntry se1("ExampleDotCom", "http://site.example.com", SearchEntry::Type::IMAGE);
    se1.addProperty(1, "aaaa");
    se1.addProperty(3, "bbbb");
    se1.addProperty(5, "cccc");
    se1.addProperty(7, "dddd");

    std::cerr << se1 << "\n";

    SdsBytesBuf bb;
    se1.write(bb);

    bb.rewind();

    SearchEntry se2("empty", "");
    se2.read(bb);

    ASSERT_EQ(se2.getSimHash().getId(), se1.getSimHash().getId());
    ASSERT_EQ(se2.getTitle(), "ExampleDotCom");
    ASSERT_EQ(se2.getUrl(), "http://site.example.com");
    ASSERT_EQ(se2.getProperty(1), "aaaa");
    ASSERT_EQ(se2.getProperty(3), "bbbb");
    ASSERT_EQ(se2.getProperty(5), "cccc");
    ASSERT_EQ(se2.getProperty(7), "dddd");
}

int main(int argc, char** argv)
{
    ::testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
