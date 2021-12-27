from setuptools import setup

setup(name='sniffDogSniff',
      version='0.5',
      description='Web Search web-scraping tool',
      author='c3rzTheFrog',
      license='GPLv3',
      packages=['sniffdogsniff'],
      install_requires=['pandas', 'requests-html', 'tqdm', 'PyQt5'],
      entry_points = {
        'console_scripts': ['sniffdogsniff.sds:main'],
      },
      zip_safe=False)