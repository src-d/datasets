# Structural features extracted from commits ![size 1.9GB](https://img.shields.io/badge/size-1.9GB-green.svg)

[Download link.](https://drive.google.com/file/d/1ouXFVfz2RG0Uj9Ljrniu3amsAFLg91fR)

JSON files with structural features of 1.6 million commits in 622 Java repositories on GitHub. Those features
are extracted from the corresponding AST differences using [Coming](https://github.com/SpoonLabs/coming).

## Format
All the commits for a particular repository are stored in a single JSON file, one object per line.
That JSON file is stored in the directory corresponding to that repository.
For example, the features of the commits in [ReactiveX/RxJava](https://github.com/ReactiveX/RxJava/) are stored in the directory `ReactiveX/RxJava`.
There is also `stats.json` in each directory with feature extraction statistics.
Finally, everything is xz-compressed. The uncompressed size is 49GB.

#### Format of the JSON objects
Each JSON object corresponds to a commit. For every modified file in that particular commit, it stores an array of edits which would produce the destination AST if applied to the source AST. Every edit contains information about the type of a change (`INS`, `DEL`, `MOV`, `UPD`), the changed entity (the types of entities are listed below), the list of parent and children nodes in the AST and the location in the file. Depending on the type of the change, some fields may be missing. For example, if the type of the change is `DEL`, the field `location_dst` — which corresponds to the location in the new version of the file — will not be present. Here is a sample pretty-printed JSON object:
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
Besides, there is a full [example.json](example.json).

#### Types of edit actions
* `INS`: Insertion of a node or a subtree in the AST
* `DEL`: Deletion of a node or a subtree in the AST
* `MOV`: Move of a node or a subtree within the AST
* `UPD`: Update of a node or a subtree in the AST

#### Types of entities
<details>
<summary>Complete list of entity types.</summary>
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

There is a plot which shows the [distribution of different types of changes](op_types.html). Note that the x-axis has a log scale. The type of a change is a composition of the type of the edit action and the type of the entity. For example, `INS/VariableRead` means that a variable access was inserted into the code.
Jupyter Notebook with code to produce this plot could be accessed [here](plot_statistics.ipynb).

## Origins
The included repositories have more than 500 stars and more than 1000 commits. We considered only the default branch. We [forked and modified](https://github.com/Jan21/coming) [Coming](https://github.com/SpoonLabs/coming). Internally, this tool uses [GumTreeDiff](https://github.com/SpoonLabs/gumtree-spoon-ast-diff) to compute the set of AST edits. Be aware that this algorithm is not perfect and in some cases may produce a few non-intuitive edits.

## License

[Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/)
