import {
    Filter,
    ReferenceArrayInput,
    SelectArrayInput,
    TextInput,
    List,
    Datagrid,
    TextField,
    SimpleForm,
    Show,
    SimpleShowLayout
} from 'react-admin';
import ReactJson from 'react-json-view';
import jp from 'jsonpath';
import { useFormState } from 'react-final-form';
import isEmpty from 'lodash/isEmpty'

const InstanceFilter = (props) => (
    <Filter {...props}>
        <TextInput label="Filter" source="filter" alwaysOn />
        <ReferenceArrayInput alwaysOn label="Backend" source="backend" reference="backends">
            <SelectArrayInput optionText="name" optionValue="name" />
        </ReferenceArrayInput>
    </Filter>
);

const jsonQuery = (raw, path) => {
    if (isEmpty(path)) return raw

    try {
        return jp.query(raw, path)
    } catch (e) {}

    return []
}

const PathInput = ({ raw }) => {
    const { values } = useFormState();
    return (
        <ReactJson name={false} src={jsonQuery(raw, values.path)} />
    );
};

const InstancePanel = ({ id, record, resource }) => (
    <div>
        <SimpleForm toolbar={false}>
            <TextInput source="path" label="JsonPath" />
            <PathInput raw={record.raw} />
        </SimpleForm>
    </div>
);

export const InstanceList = (props) => (
    <List filters={<InstanceFilter />} bulkActionButtons={false} {...props}>
        <Datagrid expand={<InstancePanel />}>
            <TextField source="id" />
            <TextField source="name" />
            <TextField source="backend_name" />
            <TextField source="private_ip" />
            <TextField source="public_ip" />
            <TextField source="status" />
            <TextField source="type" />
        </Datagrid>
    </List>
);

export const InstanceShow = (props) => (
    <Show {...props}>
        <SimpleShowLayout>
            <TextField source="name" />
        </SimpleShowLayout>
    </Show>
);