import satori from "satori";
import { Resvg } from "@resvg/resvg-js";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const interRegular = readFileSync(
  join(process.cwd(), "src/assets/fonts/Inter-Regular.ttf")
);
const interSemiBold = readFileSync(
  join(process.cwd(), "src/assets/fonts/Inter-SemiBold.ttf")
);

export async function generateOgImage(
  title: string,
  description: string
): Promise<Buffer> {
  const svg = await satori(
    {
      type: "div",
      props: {
        style: {
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          padding: "60px",
          background: "linear-gradient(135deg, #0d0d0d 0%, #0F2535 50%, #151515 100%)",
          fontFamily: "Inter",
        },
        children: [
          {
            type: "div",
            props: {
              style: {
                display: "flex",
                flexDirection: "column",
                gap: "20px",
              },
              children: [
                {
                  type: "div",
                  props: {
                    style: {
                      fontSize: "48px",
                      fontWeight: 600,
                      color: "#ffffff",
                      lineHeight: 1.2,
                      maxWidth: "900px",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                    },
                    children: title,
                  },
                },
                {
                  type: "div",
                  props: {
                    style: {
                      fontSize: "24px",
                      fontWeight: 400,
                      color: "#999999",
                      lineHeight: 1.4,
                      maxWidth: "800px",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                    },
                    children: description,
                  },
                },
              ],
            },
          },
          {
            type: "div",
            props: {
              style: {
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
              },
              children: [
                {
                  type: "div",
                  props: {
                    style: {
                      display: "flex",
                      alignItems: "center",
                      gap: "12px",
                    },
                    children: [
                      {
                        type: "div",
                        props: {
                          style: {
                            width: "40px",
                            height: "40px",
                            borderRadius: "8px",
                            background: "#5CA3CA",
                          },
                          children: "",
                        },
                      },
                      {
                        type: "div",
                        props: {
                          style: {
                            fontSize: "28px",
                            fontWeight: 600,
                            color: "#ffffff",
                          },
                          children: "Beluga AI",
                        },
                      },
                    ],
                  },
                },
                {
                  type: "div",
                  props: {
                    style: {
                      fontSize: "20px",
                      color: "#666666",
                    },
                    children: "beluga-ai.org",
                  },
                },
              ],
            },
          },
        ],
      },
    },
    {
      width: 1200,
      height: 630,
      fonts: [
        { name: "Inter", data: interRegular, weight: 400, style: "normal" as const },
        { name: "Inter", data: interSemiBold, weight: 600, style: "normal" as const },
      ],
    }
  );

  const resvg = new Resvg(svg, {
    fitTo: { mode: "width" as const, value: 1200 },
  });

  return resvg.render().asPng();
}
