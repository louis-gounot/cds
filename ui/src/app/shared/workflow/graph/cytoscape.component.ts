import {
    AfterViewInit, ApplicationRef,
    ChangeDetectionStrategy, ChangeDetectorRef,
    Component, ComponentFactoryResolver,
    Input,
    OnChanges,
    OnInit,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import * as cytoscape from 'cytoscape';
import * as dagre from 'cytoscape-dagre';
import * as klay from 'cytoscape-klay';
import { CytoscapeOptions } from 'cytoscape';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';

import { CytoscapeAngularComponentContainer, ICyEventObject } from 'app/model/cytoscape-nodehtml';


@Component({
    selector: 'app-cytoscape',
    template: '<div id="cy" #container></div>',
    styleUrls: ['./cytoscape.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class CytoscapeComponent implements OnInit, OnChanges, AfterViewInit {

    @Input() options: CytoscapeOptions;
    @Input() zoom: any;
    @Input() project: Project;
    @Input() workflow: Workflow;
    @ViewChild('container', {read: ViewContainerRef, static: true}) container: ViewContainerRef;
    private cy: cytoscape.Core;
    private ready = false;
    private angularContainer: CytoscapeAngularComponentContainer;

    public constructor(private el: ViewContainerRef, private fact: ComponentFactoryResolver, private _cd: ChangeDetectorRef, private appRef: ApplicationRef ) {
        //register(cytoscape);
        cytoscape.use(dagre);
        cytoscape.use(klay);
    }

    ngOnInit(): void {

    }

    ngAfterViewInit(): void {
        this.ready = true;
        this.render();
    }

    public ngOnChanges(): any {
        this.render();
    }

    public render() {
        if (!this.options || !this.ready) {
            return;
        }

        this.zoom = this.zoom || {
            min: 0.25,
            max: 1.5
        };
        console.log('coucou');
        if (!this.cy) {
            var dagreOpts = {
                name: 'dagre',
                // dagre algo options, uses default value on undefined
                nodeSep: 10, // the separation between adjacent nodes in the same rank
                edgeSep: 5, // the separation between adjacent edges in the same rank
                rankSep: 15, // the separation between each rank in the layout
                rankDir: 'LR', // 'TB' for top to bottom flow, 'LR' for left to right,
                ranker: undefined, // Type of algorithm to assign a rank to each node in the input graph. Possible values: 'network-simplex', 'tight-tree' or 'longest-path'
                minLen: function( edge ){ return 1; }, // number of ranks to keep between the source and target of the edge
                edgeWeight: function( edge ){ return 1; }, // higher weight edges are generally made shorter and straighter than lower weight edges

                // general layout options
                fit: true, // whether to fit to viewport
                padding: 30, // fit padding
                spacingFactor: undefined, // Applies a multiplicative factor (>0) to expand or compress the overall area that the nodes take up
                nodeDimensionsIncludeLabels: false, // whether labels should be included in determining the space used by a node
                animate: false, // whether to transition the node positions
                animateFilter: function( node, i ){ return true; }, // whether to animate specific nodes when animation is on; non-animated nodes immediately go to their final positions
                animationDuration: 500, // duration of animation in ms if enabled
                animationEasing: undefined, // easing of animation if enabled
                boundingBox: undefined, // constrain layout bounds; { x1, y1, x2, y2 } or { x1, y1, w, h }
                transform: function( node, pos ){ return pos; }, // a function that applies a transform to the final node position
                ready: function(){}, // on layoutready
                stop: function(){} // on layoutstop
            };
            var options = {
                name: 'klay',
                nodeDimensionsIncludeLabels: false, // Boolean which changes whether label dimensions are included when calculating node dimensions
                fit: true, // Whether to fit
                padding: 20, // Padding on fit
                animate: false, // Whether to transition the node positions
                animateFilter: function( node, i ){ return true; }, // Whether to animate specific nodes when animation is on; non-animated nodes immediately go to their final positions
                animationDuration: 500, // Duration of animation in ms if enabled
                animationEasing: undefined, // Easing of animation if enabled
                transform: function( node, pos ){ return pos; }, // A function that applies a transform to the final node position
                ready: undefined, // Callback on layoutready
                stop: undefined, // Callback on layoutstop
                klay: {
                    // Following descriptions taken from http://layout.rtsys.informatik.uni-kiel.de:9444/Providedlayout.html?algorithm=de.cau.cs.kieler.klay.layered
                    addUnnecessaryBendpoints: false, // Adds bend points even if an edge does not change direction.
                    aspectRatio: 1.6, // The aimed aspect ratio of the drawing, that is the quotient of width by height
                    borderSpacing: 20, // Minimal amount of space to be left to the border
                    compactComponents: false, // Tries to further compact components (disconnected sub-graphs).
                    crossingMinimization: 'LAYER_SWEEP', // Strategy for crossing minimization.
                    /* LAYER_SWEEP The layer sweep algorithm iterates multiple times over the layers, trying to find node orderings that minimize the number of crossings. The algorithm uses randomization to increase the odds of finding a good result. To improve its results, consider increasing the Thoroughness option, which influences the number of iterations done. The Randomization seed also influences results.
                    INTERACTIVE Orders the nodes of each layer by comparing their positions before the layout algorithm was started. The idea is that the relative order of nodes as it was before layout was applied is not changed. This of course requires valid positions for all nodes to have been set on the input graph before calling the layout algorithm. The interactive layer sweep algorithm uses the Interactive Reference Point option to determine which reference point of nodes are used to compare positions. */
                    cycleBreaking: 'GREEDY', // Strategy for cycle breaking. Cycle breaking looks for cycles in the graph and determines which edges to reverse to break the cycles. Reversed edges will end up pointing to the opposite direction of regular edges (that is, reversed edges will point left if edges usually point right).
                    /* GREEDY This algorithm reverses edges greedily. The algorithm tries to avoid edges that have the Priority property set.
                    INTERACTIVE The interactive algorithm tries to reverse edges that already pointed leftwards in the input graph. This requires node and port coordinates to have been set to sensible values.*/
                    direction: 'RIGHT', // Overall direction of edges: horizontal (right / left) or vertical (down / up)
                    /* UNDEFINED, RIGHT, LEFT, DOWN, UP */
                    edgeRouting: 'ORTHOGONAL', // Defines how edges are routed (POLYLINE, ORTHOGONAL, SPLINES)
                    edgeSpacingFactor: 0.5, // Factor by which the object spacing is multiplied to arrive at the minimal spacing between edges.
                    feedbackEdges: false, // Whether feedback edges should be highlighted by routing around the nodes.
                    fixedAlignment: 'NONE', // Tells the BK node placer to use a certain alignment instead of taking the optimal result.  This option should usually be left alone.
                    /* NONE Chooses the smallest layout from the four possible candidates.
                    LEFTUP Chooses the left-up candidate from the four possible candidates.
                    RIGHTUP Chooses the right-up candidate from the four possible candidates.
                    LEFTDOWN Chooses the left-down candidate from the four possible candidates.
                    RIGHTDOWN Chooses the right-down candidate from the four possible candidates.
                    BALANCED Creates a balanced layout from the four possible candidates. */
                    inLayerSpacingFactor: 1.0, // Factor by which the usual spacing is multiplied to determine the in-layer spacing between objects.
                    layoutHierarchy: false, // Whether the selected layouter should consider the full hierarchy
                    linearSegmentsDeflectionDampening: 0.3, // Dampens the movement of nodes to keep the diagram from getting too large.
                    mergeEdges: false, // Edges that have no ports are merged so they touch the connected nodes at the same points.
                    mergeHierarchyCrossingEdges: true, // If hierarchical layout is active, hierarchy-crossing edges use as few hierarchical ports as possible.
                    nodeLayering:'NETWORK_SIMPLEX', // Strategy for node layering.
                    /* NETWORK_SIMPLEX This algorithm tries to minimize the length of edges. This is the most computationally intensive algorithm. The number of iterations after which it aborts if it hasn't found a result yet can be set with the Maximal Iterations option.
                    LONGEST_PATH A very simple algorithm that distributes nodes along their longest path to a sink node.
                    INTERACTIVE Distributes the nodes into layers by comparing their positions before the layout algorithm was started. The idea is that the relative horizontal order of nodes as it was before layout was applied is not changed. This of course requires valid positions for all nodes to have been set on the input graph before calling the layout algorithm. The interactive node layering algorithm uses the Interactive Reference Point option to determine which reference point of nodes are used to compare positions. */
                    nodePlacement:'BRANDES_KOEPF', // Strategy for Node Placement
                    /* BRANDES_KOEPF Minimizes the number of edge bends at the expense of diagram size: diagrams drawn with this algorithm are usually higher than diagrams drawn with other algorithms.
                    LINEAR_SEGMENTS Computes a balanced placement.
                    INTERACTIVE Tries to keep the preset y coordinates of nodes from the original layout. For dummy nodes, a guess is made to infer their coordinates. Requires the other interactive phase implementations to have run as well.
                    SIMPLE Minimizes the area at the expense of... well, pretty much everything else. */
                    randomizationSeed: 1, // Seed used for pseudo-random number generators to control the layout algorithm; 0 means a new seed is generated
                    routeSelfLoopInside: false, // Whether a self-loop is routed around or inside its node.
                    separateConnectedComponents: true, // Whether each connected component should be processed separately
                    spacing: 20, // Overall setting for the minimal amount of space to be left between objects
                    thoroughness: 7 // How much effort should be spent to produce a nice layout..
                },
                priority: function( edge ){ return null; }, // Edges with a non-nil value are skipped when greedy edge cycle breaking is enabled
            };


            this.cy = cytoscape({
                container: this.container.element.nativeElement,
                minZoom: this.zoom.min,
                maxZoom: this.zoom.max,
                style: this.options.style,
                layout: dagreOpts, //options,//dagreOpts,
                elements: this.options.elements
            });
            //this.cy.layout(options).run();

            if (!this.angularContainer) {
                this.angularContainer = new CytoscapeAngularComponentContainer(this.cy, this.fact, this.project, this.workflow, this.el.injector, this.appRef);
                this.cy.one("render", (cy: any) => {
                    this.angularContainer.refreshData(<ICyEventObject>cy);
                    this.angularContainer.refreshView(<ICyEventObject>cy);
                });
                this.cy.on("pan zoom", (cy: any) => {
                    console.log(cy);
                    this.angularContainer.refreshView(<ICyEventObject>cy);
                });
            }
            this._cd.detectChanges();
        } else {
            /*
            this.cy.layout(this.options.layout);
            this.cy.nodes().remove();
            this.cy.add(<ElementDefinition>this.options.elements);
            this.cy.minZoom(this.zoom.min);
            this.cy.minZoom(this.zoom.max);
            this.cy.style(this.options.style);

             */
        }
    }
}

