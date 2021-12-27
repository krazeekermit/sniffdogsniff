from setuptools import setup

setup(name='sniffDogSniff',
      version='0.5',
      description='Web Search web-scraping tool',
      author='c3rzTheFrog',
      license='GPLv3',
      packages=['sniffdogsniff'],
      install_requires=['pandas', 'requests-html', 'tqdm', 'PyQt5'],
      zip_safe=False)