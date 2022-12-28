from setuptools import setup

setup(name='sniffsogsniff',
      version='1.0',
      description='P2P Web Search Engine',
      author='c3rzTheFrog',
      license='GPLv3',
      packages=['sds', 'sdsjsonrpc'],
      install_requires=[
          'requests',
          'requests-html',
          'pysocks'
      ],
      entry_points={
          'console_scripts': ['sniffdogsniff'],
      },
      zip_safe=False)
