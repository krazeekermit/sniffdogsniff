#ifndef SDSCONFIGFILE_H
#define SDSCONFIGFILE_H

#include <iostream>
#include <vector>

class SdsConfigFile
{
public:
    class Section
    {
        friend class SdsConfigFile;
    public:
        bool hasValue(const char *key);

        std::string lookupString(const char *key, const char *defaultValue = "");
        void lookupStrings(const char *key, std::vector<std::string> &list);
        bool lookupBool(const char *key, bool defaultValue = false);
        int lookupInt(const char *key, int defaultValue = 0);

        std::string getName() const;
        const std::vector<std::pair<std::string, std::string>> *values();

        friend std::ostream &operator<<(std::ostream &os, const Section *section);

    private:
        Section(const char *_name);

        bool lookupValue(const char *key, std::string &value);

        std::string name;
        std::vector<std::pair<std::string, std::string>> valuesList;
    };

    SdsConfigFile();
    ~SdsConfigFile();

    Section *getDefaultSection() const;

    bool hasSection(const char *key);
    Section *lookupSection(const char *name);
    void lookupSections(const char *name, std::vector<Section*> &list);

    void parse(const char *path);

    friend std::ostream &operator<<(std::ostream &os, const SdsConfigFile *cfg);

private:
    Section *defaultSection;
    std::vector<Section*> sections;
};

#endif // SDSCONFIGFILE_H
