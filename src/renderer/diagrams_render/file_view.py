import diagrams

from src.core.bt_module import BTModule
from src.core.policies.FilePolicies import (
    FilePolicyCantDependPolicy,
    FilePolicyMustDependPolicy,
)

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from src.core.bt_graph import BTGraph


def render(graph: "BTGraph"):
    node_map = {}
    root_module = graph.root_module

    def create_nodes(module: BTModule):
        with diagrams.Cluster(label=module.name):
            for bt_node in module.file_list:
                n = diagrams.Node(label=bt_node.label)
                node_map[bt_node.uid] = n
            for child_module in module.child_module:
                create_nodes(child_module)

    create_nodes(root_module)

    def render_module(module: BTModule):
        for bt_node in module.file_list:
            edges = bt_node.edge_to
            diagram_node = node_map[bt_node.uid]
            white_listed_nodes = [
                policy.must_depend_node
                for policy in bt_node.policies
                if isinstance(policy, FilePolicyMustDependPolicy)
            ]

            black_listed_nodes = [
                policy.cant_depend_node
                for policy in bt_node.policies
                if isinstance(policy, FilePolicyCantDependPolicy)
            ]

            for edge in edges:
                if edge.uid not in node_map:
                    continue
                edge_node = node_map[edge.uid]
                if edge.uid in [white_node.uid for white_node in white_listed_nodes]:
                    diagram_node >> diagrams.Edge(color="blue") >> edge_node
                elif edge.uid in [black_node.uid for black_node in black_listed_nodes]:
                    diagram_node >> diagrams.Edge(color="red") >> edge_node
                else:
                    diagram_node >> edge_node

        for child_module in module.child_module:
            render_module(child_module)

    render_module(root_module)
