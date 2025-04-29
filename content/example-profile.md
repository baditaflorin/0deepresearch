---
title: Example Research Profile
author: AI Assistant
date: 2025-04-28
tags: [example, go, markdown, webdev]
---

# Introduction to 0DeepResearch

This is an **example profile** created for the *0DeepResearch* application (Version 0.1). It demonstrates the use of Markdown syntax along with optional YAML front matter at the top of the file.

The front matter (the section between the `---` lines) can be used to define metadata like the `title`, `author`, `date`, or custom `tags`. Currently, only the `title` is actively used by the application to display in lists and page titles.

## Key Features Demonstrated

* Standard Markdown formatting:
    * *Italics* and **Bold** text.
    * `Inline code snippets`.
    * [Hyperlinks](https://github.com/yuin/goldmark) to external resources.
* Headings of different levels (like `#` and `##`).
* Ordered lists:
    1.  First item in the list.
    2.  Second item, demonstrating numbering.
* Unordered lists:
    * Bullet point using asterisk.
    * Another bullet point.
* Blockquotes:
  > This is a blockquote, useful for highlighting text or quotes.
* Code Blocks (using GitHub Flavored Markdown style):

    ```go
    package main

    import "fmt"

    // Simple Go example
    func main() {
        message := "Hello from the example profile!"
        fmt.Println(message)
    }
    ```bash
    # Example shell command
    echo "Markdown rendering is working!"
    go run .
    ```

## Placeholder Content

This section contains some placeholder text (Lorem Ipsum) to demonstrate how longer paragraphs are rendered.

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.

## Conclusion

This file serves as a basic demonstration and placeholder. You can:

1.  **Modify** this file directly.
2.  **Add** more `.md` files to the `content/` directory.
3.  **Upload** new `.md` files using the `/admin` page in the running application.

Remember that the filename (e.g., `example-profile.md`) determines the URL slug (`example-profile`).