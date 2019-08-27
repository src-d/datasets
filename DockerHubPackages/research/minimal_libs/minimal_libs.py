import pandas as pd
import re 
import argparse
from modelforge import Model, register_model

parser = argparse.ArgumentParser(description='Find minimal native packages required by a python package.')
parser.add_argument('-s', '--search', dest='search', action='store_const',
                    const=True, default=False,
                    help='search in supported python package names instead of computing minimal libs')
parser.add_argument('-p', '--package', metavar='package', type=str,
                    help='the name of the python package to search for')
parser.add_argument('-i', '--interactive', dest='interactive', action='store_const',
                    const=True, default=False,
                    help='run in REPL mode instead of CLI')
parser.add_argument('-q', '--quantile',
                    default=None,
                    help='specify the minimal cooccurence quantile to consider a requirement. default=0.95')

names = pd.read_csv('./python_libs.csv', header=None)[0]

libraries = pd.read_csv('./libraries_raw.csv', index_col=0)
libraries = libraries[~libraries.distro.isin(
    ['', 'photon', 'sabayon', 'distrib_id=ubuntu', 'opensuse-tumbleweed'])]

native_libs = libraries[libraries['type']
                        == 'native']

liboccurence = native_libs.groupby(["name", "distro"]).image.nunique()

nimage = native_libs.groupby(["distro"]).image.nunique()

defaultlibs = {distro: []
               for distro in list(native_libs.groupby('distro').count().index)}

for lib in liboccurence.index.get_level_values('name'):
    for distro in liboccurence.ix[lib].index:
        if liboccurence[lib, distro] == nimage[distro]:
            defaultlibs[distro] += [lib]

class MinimalPackageSet(Model):
    """
    Simple co-occurence based model that finds minimal native package set needed to run
    a python package.
    """
    NAME = "minimal-package-set"
    VENDOR = "source{d}"
    DESCRIPTION = "Model that contains a set of libraries and their occurences in docker images."
    LICENSE = "ODbL-1.0"

    def construct(self, packages: pd.DataFrame, python_package_names: pd.DataFrame, default_packages: {str: [str]}=None):
        self._packages = packages
        self._python_package_names = python_package_names
        
        if default_packages is not None:
            self._default_packages = default_packages
        else:
            default_packages = {distro: []
                        for distro in list(native_libs.groupby('distro').count().index)}

            for lib in liboccurence.index.get_level_values('name'):
                for distro in liboccurence.ix[lib].index:
                    if liboccurence[lib, distro] == nimage[distro]:
                        defaultlibs[distro] += [lib]

            self._default_packages = default_packages
        return self

    def search_by_name(name):
        """Search a python library by name

        :param name: the name of the python lib to find minimal packages for
        :type name: string
        :return: list of python packages names
        :rtype: [string]
        """
        return names[
                    names.str.contains(r'' + re.escape(name.lower()) + r'', case=False)
                ].to_dict(orient='values')

    def get_minimal_libs(python_lib, quantile=0.95):
        """Find minimal required native packages in different linux distribution for the given
        Python library

        :param python_lib: the name of the python lib to find minimal packages for
        :type python_lib: string
        :return: dictionnary of list of libraries by distribution
        :rtype: {distro-name: [string]}
        """
        lib_images = libraries[libraries['name']
                                        == python_lib]['image']
        
        co_occurences = libraries[libraries['type']
                                    == 'native']
        
        co_occurences = co_occurences[co_occurences.image.isin(lib_images)]

        quantiles = co_occurences.groupby(['name', 'distro'])[
            'image'].count().unstack().quantile(q=quantile)
        
        cooc = co_occurences.groupby(['name', 'distro'])[
            'image'].count().unstack()
        
        
        libs = {}
        # 
        for index in quantiles.index:
            libs[index] = cooc[index][cooc[index] >= quantiles[index]].index

        required_packages = {distro: set() for distro in defaultlibs}
        
        # Remove default distribution libs from needed libs
        for distro in libs:
            required_packages[distro] = set(libs[distro])-set(defaultlibs[distro])

        return required_packages


if __name__ == "__main__":
    print(pd.__version__)
    args = parser.parse_args()
    print(args)
    minimal_package_set = MinimalPackageSet().load()
    # Running in CLI mode
    if not args.interactive:
        if args.search:
            print(minimal_package_set.search_by_name(args.package))
        else: 
            try:
                minimal_package_set.get_minimal_libs(args.package)
            except Exception as e:
                print(e)
                print("Are you sure this python package exist ?")
                print("perhaps you meant",
                      minimal_package_set.search_by_name(args.package))
    # Running in REPL mode.
    else:
        while True:
            lib = str(input())
            try:
                minimal_package_set.get_minimal_libs(lib)
            except Exception as e:
                print(e)
                print("Are you sure this python package exist ?")
                print("perhaps you meant", minimal_package_set.search_by_name(lib))

