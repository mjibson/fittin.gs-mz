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
import { FitData, makeFitSummary } from './Fits';

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
			setData(data);
			setTitle(data.Ship.Name);
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
					<Link to={'/?ship=' + data.Ship.ID}>{data.Ship.Name}</Link>
					<span
						style={{ color: 'var(--emph-medium)' }}
						className="pointer fr f6 pa1"
						onClick={() => setSaved(!saved)}
					>
						[{saved ? 'un' : ''}save]
					</span>
				</h2>
				<Render id={data.Ship.ID} size={256} alt={data.Ship.Name} />
				<pre className="bg-dp04 pa1 f6">{TextFit(data)}</pre>
				{data.Cost ? (
					<div>
						Fitted value: <ISK isk={data.Cost} />
					</div>
				) : null}
				<div className="list">
					<a href={'https://zkillboard.com/kill/' + data.ID + '/'}>
						zkillboard
					</a>
					<Ref ID={data.Ship.ID} />
				</div>
			</div>
			<div className={flexChildrenClass}>
				<div>
					<h3>high slots</h3>
					<Slots items={data.Hi} />
				</div>
				<div>
					<h3>medium slots</h3>
					<Slots items={data.Med} />
				</div>
				<div>
					<h3>low slots</h3>
					<Slots items={data.Lo} />
				</div>
				<div>
					<h3>rigs</h3>
					<Slots items={data.Rig} />
				</div>
				{data.Sub[0] ? (
					<div>
						<h3>subsystems</h3>
						<Slots items={data.Sub} />
					</div>
				) : null}
				<div>
					<h3>charges</h3>
					<Slots items={data.Charge} />
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
	const fit = ['[' + data.Ship.Name + ']'];
	[data.Lo, data.Med, data.Hi, data.Rig, data.Sub].forEach((slot, idx) => {
		if (idx > 0) {
			fit.push('');
		}
		slot.forEach((v) => {
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
