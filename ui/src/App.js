import * as React from "react";
import { Admin, Resource } from 'react-admin';
import jsonServerProvider from 'ra-data-json-server';
import InstanceIcon from '@material-ui/icons/Book';
import BackendIcon from '@material-ui/icons/Satellite';
import { SearchList } from './search';
import { BackendList } from './backends';

const dataProvider = jsonServerProvider('api/v1');
const App = () => (
    <Admin dataProvider={dataProvider} logoutButton={null}>
        <Resource sort={null} options={{ label: 'Instances' }} name="search" list={SearchList} icon={InstanceIcon} />
        <Resource sort={null} name="backends" list={BackendList} icon={BackendIcon} />
    </Admin>
);

export default App;