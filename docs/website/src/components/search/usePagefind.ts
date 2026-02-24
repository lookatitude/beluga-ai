import { useState, useCallback, useRef, useEffect } from "react";

export interface SearchResult {
  id: string;
  url: string;
  title: string;
  excerpt: string;
  section: string;
  subResults: Array<{
    url: string;
    title: string;
    excerpt: string;
  }>;
}

interface PagefindInstance {
  search: (
    query: string,
    options?: { filters?: Record<string, string> }
  ) => Promise<{
    results: Array<{
      id: string;
      data: () => Promise<{
        url: string;
        meta: { title?: string };
        excerpt: string;
        filters: Record<string, string[]>;
        sub_results: Array<{
          url: string;
          title: string;
          excerpt: string;
        }>;
      }>;
    }>;
  }>;
  filters: () => Promise<Record<string, Record<string, number>>>;
}

export function usePagefind() {
  const [results, setResults] = useState<SearchResult[]>([]);
  const [sections, setSections] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [resultCount, setResultCount] = useState(0);
  const pagefindRef = useRef<PagefindInstance | null>(null);
  const abortRef = useRef(0);

  const getPagefind = useCallback(async (): Promise<PagefindInstance | null> => {
    if (pagefindRef.current) return pagefindRef.current;
    try {
      const basePath =
        import.meta.env.BASE_URL.replace(/\/$/, "") + "/pagefind/pagefind.js";
      const pf = await import(/* @vite-ignore */ basePath);
      await pf.init?.();
      pagefindRef.current = pf;
      return pf;
    } catch {
      console.error("Failed to load Pagefind");
      return null;
    }
  }, []);

  useEffect(() => {
    (async () => {
      const pf = await getPagefind();
      if (!pf) return;
      const filters = await pf.filters();
      if (filters.section) {
        setSections(Object.keys(filters.section).sort());
      }
    })();
  }, [getPagefind]);

  const search = useCallback(
    async (query: string, section?: string) => {
      if (!query.trim()) {
        setResults([]);
        setResultCount(0);
        setLoading(false);
        return;
      }

      setLoading(true);
      const searchId = ++abortRef.current;

      const pf = await getPagefind();
      if (!pf || searchId !== abortRef.current) return;

      const filters = section ? { filters: { section } } : undefined;
      const response = await pf.search(query, filters);

      if (searchId !== abortRef.current) return;

      setResultCount(response.results.length);

      const loaded = await Promise.all(
        response.results.slice(0, 10).map(async (r) => {
          const data = await r.data();
          return {
            id: r.id,
            url: data.url,
            title: data.meta?.title || data.url,
            excerpt: data.excerpt,
            section: data.filters?.section?.[0] || "Docs",
            subResults: data.sub_results || [],
          };
        })
      );

      if (searchId !== abortRef.current) return;

      setResults(loaded);
      setLoading(false);
    },
    [getPagefind]
  );

  return { results, sections, loading, resultCount, search };
}
