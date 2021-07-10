import React, { useState, Fragment, useEffect } from 'react';
import { Link, useLocation, useHistory } from 'react-router-dom';
import SortedTable from './SortedTable';
import {
	Ref,
	Icon,
	ISK,
	setTitle,
	Fetch,
	flexChildrenClass,
	ItemCharge,
} from './common';

function addParam(search: URLSearchParams, name: string, val: string) {
	// Prevent duplicates.
	if (search.getAll(name).includes(val)) {
		return search;
	}
	search.append(name, val);
	return search;
}

function removeParam(search: string, name: string, val: string) {
	const old = new URLSearchParams(search);
	const next = new URLSearchParams();
	for (let pair of old.entries()) {
		if (pair[0] === name && pair[1] === val) {
			continue;
		}
		next.append(pair[0], pair[1]);
	}
	return next;
}

function makeURL(search: URLSearchParams) {
	const pathname = window.location.pathname;
	if (!search) {
		return pathname;
	}
	return pathname + '?' + search;
}

export default function Fits() {
	const [data, setData] = useState<FitsData | null>(null);
	const location = useLocation();
	const history = useHistory();
	setTitle();

	useEffect(() => {
		const search = location.search;
		Fetch<FitsData>('Fits' + location.search, (data) => {
			if (window.location.search !== search) {
				return;
			}
			if (data.Fits) {
				data.Fits.forEach(populateFitData);
			}
			setTitle();
			setData(data);
		});
	}, [location]);

	if (!data) {
		return <Fragment>loading...</Fragment>;
	}

	return (
		<div className="flex flex-column">
			{Object.keys(data.Filter).length ? (
				<div className={flexChildrenClass}>
					{Object.entries(data.Filter).map(([type, items]) =>
						items.map((item) => (
							<div key={item.ID} className="ma1">
								filter by {type}: {item.Name}
								{item.Group ? (
									<Link
										to={makeURL(
											addParam(
												removeParam(
													window.location.search,
													type,
													item.ID.toString()
												),
												'group',
												item.Group.toString()
											)
										)}
										className="ma1"
									>
										(group)
									</Link>
								) : null}
								<button
									className="mh2 ba b--secondary bg-dp08 pointer"
									onClick={() =>
										history.push(
											window.location.pathname +
												'?' +
												removeParam(location.search, type, item.ID.toString())
										)
									}
								>
									x
								</button>
								<Ref ID={item.ID} />
							</div>
						))
					)}
				</div>
			) : null}
			<div className={flexChildrenClass}>
				<FitsTable data={(data.Fits || []).map(makeFitSummary)} />
			</div>
		</div>
	);
}

export function FitsTable(props: { data: Array<FitSummary> }) {
	return (
		<SortedTable
			name="fits"
			sort="Cost"
			headers={[
				{
					name: 'Name',
					header: 'ship',
					cell: (_: any, row: any) => (
						<Link
							to={makeURL(
								addParam(
									new URLSearchParams(window.location.search),
									'ship',
									row.Ship
								)
							)}
							style={{ whiteSpace: 'nowrap' }}
						>
							<Icon id={row.Ship} alt={row.Name} overrideSize={32} />
							{row.Name}
						</Link>
					),
				},
				{
					name: 'Cost',
					header: 'fit',
					desc: true,
					cell: (v: any, row: any) => (
						<div style={{ textAlign: 'right' }}>
							<Link to={'/fit/' + row.Killmail}>
								{v > 0 ? <ISK isk={v} /> : 'unknown value'}
							</Link>
						</div>
					),
				},
				{
					name: 'Hi',
					header: 'high slots',
					cell: (v: any) => <SlotSummary items={v} />,
					desc: true,
					cmp: slotCmp,
				},
				{
					name: 'Med',
					header: 'med slots',
					cell: (v: any) => <SlotSummary items={v} />,
					desc: true,
					cmp: slotCmp,
				},
				{
					name: 'Lo',
					header: 'low slots',
					cell: (v: any) => <SlotSummary items={v} />,
					desc: true,
					cmp: slotCmp,
				},
			]}
			data={props.data}
			tableClass="collapse"
			tdClass="ph2"
		/>
	);
}

function SlotSummary(props: { items: ItemCharge[] }) {
	if (!props.items) {
		return null;
	}
	const counts: { [name: string]: number } = {};
	const ids: { [name: string]: number } = {};
	props.items.forEach((v) => {
		if (!v.Name) {
			return;
		}
		if (!counts[v.Name]) {
			counts[v.Name] = 0;
			ids[v.Name] = v.ID;
		}
		counts[v.Name] += 1;
	});
	const arr = Object.entries(counts);
	arr.sort((a, b) => {
		const n = b[1] - a[1];
		if (n !== 0) {
			return n;
		}
		return a[0].localeCompare(b[0]);
	});
	return (
		<Fragment>
			{arr.map(([name, count]) => (
				<span key={name} title={name} style={{ whiteSpace: 'nowrap' }}>
					<Link
						to={makeURL(
							addParam(
								new URLSearchParams(window.location.search),
								'item',
								ids[name].toString()
							)
						)}
						style={{ color: 'var(--emph-high)', textDecoration: 'none' }}
					>
						{count}x
						<Icon id={ids[name]} alt={name} overrideSize={32} />
					</Link>
				</span>
			))}
		</Fragment>
	);
}

interface FitsData {
	Filter: {
		item: ItemCharge[];
		ship: ItemCharge[];
	};
	Fits: FitData[];
}

export interface FitData {
	// Populated from web request.
	Killmail: number;
	Ship: number;
	Cost: number;
	Names: {
		[key: string]: {
			id: number;
			name: string;
			category: string;
			group: number;
			group_name: string;
			slot: string | null;
		};
	};
	Items: {
		item_type_id: number;
		flag: number;
	}[];

	// Computed.
	Slots: {
		[key: string]: ItemCharge[];
	};
	ShipName: string;
}

export function populateFitData(obj: FitData) {
	obj.Slots = {};
	['lo', 'med', 'hi', 'rig', 'sub'].forEach((slot) => {
		obj.Slots[slot] = Array(8);
	});
	if (!obj.Items || !obj.Names[obj.Ship]) {
		obj.ShipName = 'TODO';
		return;
	}
	obj.Items.forEach((item) => {
		const id = item.item_type_id;
		const flag = item.flag;
		if (!obj.Names[id]) {
			return;
		}
		const { name, group, slot, category } = obj.Names[id];
		if (!slot) {
			return;
		}
		let offset;
		if (flag >= 27 && flag <= 34) {
			offset = 27;
		} else if (flag >= 19 && flag <= 26) {
			offset = 19;
		} else if (flag >= 11 && flag <= 18) {
			offset = 11;
		} else if (flag >= 92 && flag <= 99) {
			offset = 92;
		} else if (flag >= 125 && flag <= 132) {
			offset = 125;
		} else {
			return;
		}
		const slot_idx = flag - offset;
		if (!obj.Slots[slot][slot_idx]) {
			obj.Slots[slot][slot_idx] = {
				ID: 0,
				Name: '',
				Group: 0,
				Charge: undefined,
			};
		}
		if (category === 'charge') {
			obj.Slots[slot][slot_idx].Charge = {
				ID: id,
				Name: name,
			};
		} else {
			obj.Slots[slot][slot_idx].ID = id;
			obj.Slots[slot][slot_idx].Name = name;
			obj.Slots[slot][slot_idx].Group = group;
		}
	});
	obj.ShipName = obj.Names[obj.Ship].name;
}

export function makeFitSummary(fit: FitData): FitSummary {
	const summary: FitSummary = {
		Killmail: fit.Killmail,
		Ship: fit.Ship,
		Name: fit.ShipName,
		Cost: fit.Cost,
		Hi: fit.Slots['hi'].filter(populatedItem),
		Med: fit.Slots['med'].filter(populatedItem),
		Lo: fit.Slots['lo'].filter(populatedItem),
	};
	return summary;
}

function populatedItem(item: ItemCharge) {
	return item.Name;
}

export interface FitSummary {
	Killmail: number;
	Ship: number;
	Name: string;
	Cost: number;
	Hi: ItemCharge[];
	Med: ItemCharge[];
	Lo: ItemCharge[];
}

function slotCmp(a: any, b: any): number {
	return (a || []).length - (b || []).length;
}
