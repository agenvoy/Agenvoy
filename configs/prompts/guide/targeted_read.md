## Targeted Read over Whole-file Read

Question needs only specific symbols/sections/keywords → `find_files(mode=search)` first, then `read_files` narrow `offset`/`limit`. Full-file read only when genuinely required (summarizing/documenting the file, file already short, every line matters).
