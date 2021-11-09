# atlassian
Utilities for [Atlassian](https://www.atlassian.com/) products

## JiraD
JiraD turns Jira issue relationships from a CSV export into [PlantUML](https://www.plantuml.com/) Object Model syntax.

### Usage

    JiraD.exe [OPTION] ...

### Options
* **-in** _filename_ - Input Jira search results as comma-separated file. Defaults to 'tickets.csv'. 
* **-out** _filename_ - Output PlantUML object model syntax. Defaults to 'tickets.txt'.
* **-supplemental** _filename_ - Optional second search results input file. Can be hand-crafted. Useful for showing relationships with tickets from external Jira instances.
* **-hideSummary**=_BOOL_ = If 'true', doesn't show ticket summaries. Defaults to 'false'.
* **-hideOrphans**=_BOOL_ = If 'true', only shows tickets with relationships. Defaults to 'true'.
* **-hideKeys**=_LIST_ = Comma-separated list of issue keys to exclude from the output. Handy for eliminating noise.
* **-wrapWidth**=_NUMBER_ = Point at which to start wrapping summary text. This is an undocumented feature of PlantUML; I'm not sure of the units, but it might be pixels when images are created? Defaults to 150. 

### Notes
* Relies on the following input field names:
  * Issue key
  * Summary
  * Status
  * Inward issue link (Blocks)
  * Outward issue link (Blocks)
* Overwrites output file if it already exists

### Generate a diagram
To generate a PlantUML diagram from an output file, follow these steps.

#### SVG
    java -jar plantuml.jar -tsvg {filename}

#### PNG
    java -jar plantuml.jar {filename}

Full PlantUML command line documentation is [here](https://plantuml.com/command-line). Alternative ways
to run PlantUML can be found [here](https://plantuml.com/running).