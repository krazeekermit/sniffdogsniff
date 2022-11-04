from setuptools import setup

setup(name='sniffsogsniff',
      version='0.5',
      description='P2P Web Search Engine',
      author='c3rzTheFrog',
      license='GPLv3',
      packages=['sds', 'sdsjsonrpc'],
      install_requires=['pandas', 'tqdm'],
      entry_points = {
        'console_scripts': ['sds.sds:main'],
      },
      zip_safe=False)