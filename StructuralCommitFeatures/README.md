# Structural features extracted from commits ![size 3.2GB](https://img.shields.io/badge/size-3.2GB-green.svg)
## Dataset description
[The dataset](https://drive.google.com/open?id=1T9ICNPj0vcNnOMtWzZskhqqDD0JOHGFe) contains JSON objects with structural features for 1.6 million commits from 622 Java repositories. All commits for a particular repository are stored in JSON file (one JSON object per line). This zipped JSON file is in turn stored in a folder corresponding to that repository. 

You can download the dataset from this [link](https://drive.google.com/open?id=1T9ICNPj0vcNnOMtWzZskhqqDD0JOHGFe)

#### Folder structure
The folder structure of this dataset reflects the paths of repositories on GitHub. For example, features for commits of the repository [github.com/ReactiveX/RxJava/](https://github.com/ReactiveX/RxJava/) are stored in a zip file in a folder ReactiveX/RxJava. The zip file contains a file with one JSON object per line. Every JSON object corresponds to one commit in that repository. 

#### Format of the JSON objects
The JSON object corresponds to a commit. For every modified file in that particular commit, it stores an array of edits which would produce the destination AST when applied to the source AST. Every edit contains information about a type of change (`INS`, `DEL`, `MOV`, `UPD`), the entity changed (types of entities are listed below), list of parent and children nodes in the AST and a location within the file. Depending on the type of change, some field may not be present. For example, if the type of change is `DEL`, the field `location_dst` — which corresponds to a location within the new version of a file — will not be present. Here is the structure of the JSON object:
```json
{  
   "id":"hash of the commit",
   "files":[  
      {  
         "file_name":"name of the modified file",
         "features":[  
            {  
               "label":"label of the modified element, i.e. java.util.Map$Entry#getKey()",
               "type":"type of the modified element, i.e. Invocation",
               "op":"short name of the edit action, i.e. INS, DEL, MOV, UPD",
               "children":"Json representation of the AST subtree corresponding to this element",
               "location_src":[  
                  "number of starting line",
                  "number of ending line",
                  "number of starting character",
                  "number of ending character"
               ],
               "location_dst":"same as for location_src but w.r.t. the file after the
                               changes",
               "parents_src":{  
                  "parent_ids":"array of ids of parent nodes in the source AST; could be
                                used for matching the changes, i.e. some element may have
                                been deleted from a subtree which was moved; it's ordered
                                from the immediate parent up to the root",
                  "parent_names":"array of names of parent nodes; same order as for
                                parent_ids"
               },
               "parents_dst":"same as for parents_src but w.r.t. the AST after the changes",
               "upd_to_tree":"present only in the case of UPD action. This field contains
                              a Json representation of the resulting AST subtree which
                              correspond to the element updated"
            ]
         }
      ]
   }
]
}
```
Here is a full [example](example.json)

#### Types of edit actions
* `INS`: Insertion of a node or a subtree in the AST
* `DEL`: Deletion of a node or a subtree in the AST
* `MOV`: Move of a node or a subtree within the AST
* `UPD`: Update of a node or a subtree in the AST

#### Types of entities
<details>
<summary>Complete list of entity types</summary>
<ul>
<li>`Annotation`</li>	 
<li>`AnnotationFieldAccess`</li>	 
<li>`ArrayAccess`</li>	 
<li>`ArrayRead`</li>	 
<li>`ArrayWrite`</li>	 
<li>`Assert`</li>	 
<li>`Assignment`</li>	 
<li>`BinaryOperator`</li>	 
<li>`Block`</li>	 
<li>`Case`</li>	 
<li>`Catch`</li>	 
<li>`CatchVariableImpl`</li>	 
<li>`CFlowBreak`</li>	 
<li>`CodeSnippetExpression`</li>	 
<li>`Comment`</li>	 
<li>`Conditional`</li>	 
<li>`Constructor`</li>	 
<li>`ConstructorCall`</li>	 
<li>`Do`</li>	 
<li>`Enum`</li>	 
<li>`EnumValue`</li>	 
<li>`Field`</li>	 
<li>`FieldAccess`</li>	 
<li>`FieldRead`</li>	 
<li>`FieldWrite`</li>	 
<li>`For`</li>	 
<li>`ForEach`</li>	 
<li>`If`</li>	 
<li>`Import`</li>	 
<li>`Interface`</li>	 
<li>`Invocation`</li>	 
<li>`JavaDoag`</li>	 
<li>`LabelledFlowBreak`</li>	 
<li>`Lambda`</li>	 
<li>`Literal`</li>	 
<li>`LocalVariable`</li>	 
<li>`Method`</li>	 
<li>`NewArray`</li>	 
<li>`NewClass`</li>	 
<li>`OperatorAssignment`</li>	 
<li>`Parameter`</li>	 
<li>`Return`</li>	 
<li>`SuperAccess`</li>	 
<li>`Synchronized`</li>	 
<li>`TargetedExpression`</li>	 
<li>`ThisAccess`</li>	 
<li>`Throw`</li>	 
<li>`Try`</li>	 
<li>`TryWithResource`</li>	 
<li>`Type`</li>	 
<li>`TypeAccess`</li>	 
<li>`TypeMember`</li>	 
<li>`UnaryOperator`</li>	 
<li>`VariableRead`</li>	 
<li>`VariableWrite`</li>	 
<li>`While`</li>	 
</ul>
</details>

Here is a plot showing a [distribution of different types of changes](op_types.html). Note that the x-axis is in a log scale. Type of change is a composition of a type of the edit action and a type of the entity (for example, INS/VariableRead means that variable access was inserted into the code).
Jupyter Notebook with code to produce this plot could be accessed [here](plot_statistics.ipynb).

## Origins
The repositories were chosen based on a number of stars (>500) and number of commits (>1000).
To extract the features for all commits within a default branch in a repository, we forked and modified a tool called [Coming](https://github.com/SpoonLabs/coming). Internally, this tool uses [GumTreeDiff](https://github.com/SpoonLabs/gumtree-spoon-ast-diff) to compute the set of edits. Be aware that this algorithm is not perfect and in some cases may produce few non-intuitive edits.


## License

[Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/)
