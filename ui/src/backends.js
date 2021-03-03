import { List, Datagrid, TextField } from 'react-admin';

export const BackendList = (props) => (
    <List pagination={null} bulkActionButtons={false} {...props}>
        <Datagrid>
            <TextField source="name" />
            <TextField source="type" />
        </Datagrid>
    </List>
);