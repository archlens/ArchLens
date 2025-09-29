using SyntaxTreeManualTraversal.Model;

namespace SyntaxTreeManualTraversal
{
    static class GraphToJsonConverter
    {
        public static string ConvertToJson(DependencyGraph graph, string title)
        {
            var packagestr = "";
            for (int i = 0; i < graph.packages().Count; i++)
            {
                var package = graph.packages()[i];

                if (packagestr.Contains(package)) continue;

                var comma = "";
                if (i < graph.packages().Count - 2) comma = ",";

                packagestr +=
                    $$"""
                    
                    {
                        "name": "{{package}}",
                        "state": "NEUTRAL"
                    }{{comma}}

                """;
            }

            var str =
            $$"""
            {
                "title": "{{title}}",
                "packages": [
                    {{packagestr}}
                ],

                "edges": [
                {{graph.ToJson()}}

                ]
            }
            """;
            return str;
        }
    }
}