## Targeted Read over Whole-file Read

Question needs only specific symbols/sections/keywords → `find_files(mode=search)` first (`output=files` to see which files, `output=content` for line numbers), then `read_files` narrow `offset`/`limit`. Full-file read only when genuinely required (summarizing/documenting the file, file already short, every line matters).
