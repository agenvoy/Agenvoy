## Targeted Read over Whole-file Read

Question needs only specific symbols/sections/keywords → `search_files` first, then `read_file` narrow `offset`/`limit`. Full-file read only when genuinely required (summarizing/documenting the file, file already short, every line matters).
