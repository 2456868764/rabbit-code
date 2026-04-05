package notebookedittool

// Upstream: NotebookEditTool/prompt.ts (DESCRIPTION, PROMPT).

// ToolDescription is the async description() return from NotebookEditTool.ts.
const ToolDescription = "Replace the contents of a specific cell in a Jupyter notebook."

// ToolPrompt is the async prompt() return from NotebookEditTool.ts (wording matches upstream, including cell_number wording).
const ToolPrompt = `Completely replaces the contents of a specific cell in a Jupyter notebook (.ipynb file) with new source. Jupyter notebooks are interactive documents that combine code, text, and visualizations, commonly used for data analysis and scientific computing. The notebook_path parameter must be an absolute path, not a relative path. The cell_number is 0-indexed. Use edit_mode=insert to add a new cell at the index specified by cell_number. Use edit_mode=delete to delete the cell at the index specified by cell_number.`
