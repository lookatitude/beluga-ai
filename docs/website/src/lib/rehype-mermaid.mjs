import { visit } from "unist-util-visit";

/**
 * A lightweight rehype plugin that converts mermaid code blocks into
 * <pre class="mermaid"> elements for client-side rendering by mermaid.js.
 *
 * Transforms: <pre><code class="language-mermaid">graph LR ...</code></pre>
 * Into:       <pre class="mermaid">graph LR ...</pre>
 */
export default function rehypeMermaid() {
  return (tree) => {
    visit(tree, "element", (node, index, parent) => {
      if (
        node.tagName !== "pre" ||
        !node.children ||
        node.children.length !== 1
      )
        return;

      const code = node.children[0];
      if (
        code.type !== "element" ||
        code.tagName !== "code" ||
        !code.properties?.className
      )
        return;

      const classes = Array.isArray(code.properties.className)
        ? code.properties.className
        : [code.properties.className];

      if (!classes.includes("language-mermaid")) return;

      // Extract the text content from the code element
      const text = code.children
        .filter((c) => c.type === "text")
        .map((c) => c.value)
        .join("");

      // Replace <pre><code class="language-mermaid">...</code></pre>
      // with <pre class="mermaid">...</pre>
      node.children = [{ type: "text", value: text }];
      node.properties = { className: ["mermaid"] };
    });
  };
}
