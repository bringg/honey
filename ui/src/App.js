import * as React from "react";
import { Admin, Resource, AppBar, Layout } from 'react-admin';
import jsonServerProvider from 'ra-data-json-server';
import InstanceIcon from '@material-ui/icons/Book';
import BackendIcon from '@material-ui/icons/Satellite';
import { InstanceList, InstanceShow } from './instances';
import { BackendList } from './backends';

const dataProvider = jsonServerProvider('api/v1');

const CustomAppBar = props => <AppBar {...props} userMenu={false} />;
const CustomLayout = props => <Layout {...props} appBar={CustomAppBar} />;

const App = () => (
    <Admin dataProvider={dataProvider} layout={CustomLayout}>
        <Resource sort={null} name="instances" list={InstanceList} show={InstanceShow} icon={InstanceIcon} />
        <Resource sort={null} name="backends" list={BackendList} icon={BackendIcon} />
    </Admin>
);

export default App;