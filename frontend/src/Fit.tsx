import React, { Fragment, useState, useEffect } from 'react';
import {
	Render,
	Ref,
	ISK,
	Icon,
	ItemCharge,
	setTitle,
	Fetch,
	flexChildrenClass,
	savedPrefix,
} from './common';
import { Link, useParams } from 'react-router-dom';
import { FitData, populateFitData, makeFitSummary } from './Fits';

export default function Fit() {
	const { id } = useParams();
	const savedCookie = savedPrefix + id;

	const [data, setData] = useState<FitData | null>(null);
	const [saved, setSaved] = useState<boolean>(
		!!localStorage.getItem(savedCookie)
	);

	useEffect(() => {
		setTitle();
		Fetch<FitData>('Fit?id=' + id, (data) => {
			populateFitData(data);
			setData(data);
			setTitle(data.ShipName);
		});
	}, [id]);

	useEffect(() => {
		if (!data) {
			return;
		}
		if (!saved) {
			localStorage.removeItem(savedCookie);
		} else {
			const s = makeFitSummary(data);
			localStorage.setItem(savedCookie, JSON.stringify(s));
		}
	}, [data, saved, savedCookie]);

	if (!data) {
		return <Fragment>loading...</Fragment>;
	}

	return (
		<div className="flex flex-wrap">
			<div className={flexChildrenClass}>
				<h2>
					<Link to={'/?ship=' + data.Ship}>{data.ShipName}</Link>
					<span
						style={{ color: 'var(--emph-medium)' }}
						className="pointer fr f6 pa1"
						onClick={() => setSaved(!saved)}
					>
						[{saved ? 'un' : ''}save]
					</span>
				</h2>
				<Render id={data.Ship} size={256} alt={data.ShipName} />
				<pre className="bg-dp04 pa1 f6">{TextFit(data)}</pre>
				{data.Cost ? (
					<div>
						Fitted value: <ISK isk={data.Cost} />
					</div>
				) : null}
				<div className="list">
					<a href={'https://zkillboard.com/kill/' + data.Killmail + '/'}>
						zkillboard
					</a>
					<Ref ID={data.Ship} />
				</div>
			</div>
			<div className={flexChildrenClass}>
				<div>
					<h3>high slots</h3>
					<Slots items={data.Slots['hi']} />
				</div>
				<div>
					<h3>medium slots</h3>
					<Slots items={data.Slots['med']} />
				</div>
				<div>
					<h3>low slots</h3>
					<Slots items={data.Slots['lo']} />
				</div>
				<div>
					<h3>rigs</h3>
					<Slots items={data.Slots['rig']} />
				</div>
				{data.Slots['sub'][0] ? (
					<div>
						<h3>subsystems</h3>
						<Slots items={data.Slots['sub']} />
					</div>
				) : null}
				<div>
					<h3>charges</h3>
					<Slots items={data.Slots['charges']} />
				</div>
			</div>
		</div>
	);
}

function Slots(props: { items: ItemCharge[] }) {
	if (!props.items) {
		return null;
	}
	return (
		<Fragment>
			{props.items
				.filter((v) => v.Name)
				.map((v, idx) => {
					return (
						<div key={idx}>
							<Link to={'/?item=' + v.ID}>
								<Icon id={v.ID} alt={v.Name} />
								{v.Name}
							</Link>
						</div>
					);
				})}
		</Fragment>
	);
}

function TextFit(data: FitData) {
	const fit = ['[' + data.Names[data.Ship].name + ']'];
	['lo', 'med', 'hi', 'rig', 'sub'].forEach((slot, idx) => {
		if (idx > 0) {
			fit.push('');
		}
		data.Slots[slot].forEach((v) => {
			if (!v.Name) {
				return;
			}
			let n = v.Name;
			if (v.Charge) {
				n += ', ' + v.Charge.Name;
			}
			fit.push(n);
		});
	});
	return fit.join('\n');
}
