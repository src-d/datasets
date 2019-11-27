using System;
using System.Collections.Generic;
using dnlib.DotNet;

namespace dnlib.Examples {
	public class Example1 {
		public static void Run(string[] args) {
			var filename = args[0];
			var mod = ModuleDefMD.Load(filename);
			var namespaces = new Dictionary<string, int>();
			foreach (var type in mod.Types) {
			    if (type.Namespace == "" || type.Namespace == "System" || type.Namespace.StartsWith("System.")) {
			        continue;
			    }
			    if (!namespaces.ContainsKey(type.Namespace)) {
			        namespaces.Add(type.Namespace, 0);
			    }
				namespaces[type.Namespace]++;
			}
			foreach (var pair in namespaces) {
			    Console.WriteLine("{0}\t{1}", pair.Key, pair.Value);
			}
		}
	}
}
