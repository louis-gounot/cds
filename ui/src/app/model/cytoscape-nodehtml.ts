import * as cytoscape from 'cytoscape';
import { WorkflowWNodeComponent } from 'app/shared/workflow/wnode/wnode.component';
import { Workflow } from 'app/model/workflow.model';
import { Project } from 'app/model/project.model';
import { ApplicationRef, ComponentFactoryResolver, EmbeddedViewRef, Injector } from '@angular/core';

export type IHAlign = "left" | "center" | "right";
export type IVAlign = "top" | "center" | "bottom";

export interface ICyEventObject {
    cy: any;
    type: string;
    target: any;
}

export interface CytoscapeNodeHtmlParams {
    query?: string;
    halign?: IHAlign;
    valign?: IVAlign;
    halignBox?: IHAlign;
    valignBox?: IVAlign;
    cssClass?: string;
}

export interface HashTableElements {
    [key: string]: CytoscapeAngularComponent;
}

export interface ICytoscapeNodeHtmlPosition {
    x: number;
    y: number;
    w: number;
    h: number;
}

export class CytoscapeAngularComponentContainer {

    cy: cytoscape.Core;
    htmlContainer: HTMLDivElement;
    workflow: Workflow;
    project: Project;
    factoryResolver: ComponentFactoryResolver;
    injector: Injector;
    appRef: ApplicationRef;
    _elements: HashTableElements;

    constructor(c: cytoscape.Core, factoryResolver: ComponentFactoryResolver, project: Project, workflow: Workflow, inj: Injector, appRef: ApplicationRef) {
        this.cy = c;
        this.factoryResolver = factoryResolver;
        this.project = project;
        this.workflow = workflow;
        this.injector = inj;
        this.appRef = appRef;
        this._elements = <HashTableElements>{};

        this.init();
    }

    init(): void {
        this.htmlContainer = document.createElement('div');
        let stl = this.htmlContainer.style;
        stl.position = 'absolute';
        stl['z-index'] = 10;
        stl.width = '500px';
        stl['pointer-events'] = 'none';
        stl.margin = '0px';
        stl.padding = '0px';
        stl.border = '0px';
        stl.outline = '0px';
        stl.outline = '0px';
        this.cy.container().childNodes.item(0).appendChild(this.htmlContainer);
    }

    refreshView({cy}: ICyEventObject): void {
        const val = `translate(${cy.pan().x}px,${cy.pan().y}px) scale(${cy.zoom()})`;
        const stl = <any>this.htmlContainer.style;
        const origin = "top left";
        stl.webkitTransform = val;
        stl.msTransform = val;
        stl.transform = val;
        stl.webkitTransformOrigin = origin;
        stl.msTransformOrigin = origin;
        stl.transformOrigin = origin;
    }

    refreshData({cy}: ICyEventObject): void {
        cy.nodes().forEach( n => {
            this.addOrUpdateElem(n.id(), n.data(), {w: n.width(), h: n.height(), x: n.position('x'), y: n.position('y')});
            //this.addOrUpdateElem(n.id(), n.data(), {w: 180, h: 60, x: n.position('x'), y: n.position('y')});
        });
    }

    addOrUpdateElem(id: string, data: cytoscape.NodeDataDefinition, pos?: ICytoscapeNodeHtmlPosition) {
        let cur = this._elements[id];
        if (cur) {
            cur.updateParams();
            cur.updatePosition(pos);
        } else {
            // Create a factory to build WorkflowWNodeComponent
            let nodeComponentFactory = this.factoryResolver.resolveComponentFactory(WorkflowWNodeComponent);
            // Create the component
            let componentRef = nodeComponentFactory.create(this.injector);
            componentRef.instance.node = data['node'];
            componentRef.instance.workflow = this.workflow;
            componentRef.instance.project = this.project;
            componentRef.location.nativeElement.style.width = '180px';
            componentRef.location.nativeElement.style.height = '60px';
            // Attach component to angular tree
            this.appRef.attachView(componentRef.hostView);

            // Attach the DOM
            const domElt = (componentRef.hostView as EmbeddedViewRef<any>).rootNodes[0] as HTMLElement;
            this.htmlContainer.appendChild(domElt);

            this._elements[id] = new CytoscapeAngularComponent(domElt, data, pos);
        }
    }
}

export class CytoscapeAngularComponent {
    dom: HTMLElement;
    data: cytoscape.NodeDataDefinition;
    position: ICytoscapeNodeHtmlPosition;

    _align: [number, number, number, number];
    _position: [number,number];

    constructor(dom: HTMLElement, data: cytoscape.NodeDataDefinition, pos: ICytoscapeNodeHtmlPosition) {
        this.dom = dom;
        this.data = data;

        this.updateParams();
        this.initStyles('');
        if (pos) {
            this.updatePosition(pos);
        }
    }

    updateParams() {
        let halign = "center",
            valign = "center",
            halignBox = "center",
            valignBox = "center";
        const _align = {
            "top": -.5,
            "left": -.5,
            "center": 0,
            "right": .5,
            "bottom": .5
        };

        this._align = [
            _align[halign],
            _align[valign],
            100 * (_align[halignBox] - 0.5),
            100 * (_align[valignBox] - 0.5)
        ];
    }

    updatePosition(pos: ICytoscapeNodeHtmlPosition) {
        const prev = this._position;
        const x = pos.x + this._align[0] * pos.w;
        const y = pos.y + this._align[1] * pos.h;

        if (!prev || prev[0] !== x || prev[1] !== y) {
            this._position = [x, y];

            let valRel = `translate(${this._align[2]}%,${this._align[3]}%) `;
            let valAbs = `translate(${x.toFixed(2)}px,${y.toFixed(2)}px) `;
            let val = valRel + valAbs;
            let stl = <any>this.dom.style;
            stl.webkitTransform = val;
            stl.msTransform = val;
            stl.transform = val;
        }
    }

    initStyles(cssClass: string) {
        let stl = this.dom.style;
        stl.position = 'absolute';
        if (cssClass && cssClass.length) {
            this.dom.classList.add(cssClass);
        }
    }
}
